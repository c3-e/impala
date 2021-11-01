import java.net.URI
import java.time._
import java.time.format._
import org.apache.hadoop.fs._
import org.apache.hadoop.conf.Configuration
import sys.process._

def path = sys.env.get("WASB_PATH").getOrElse("wasb://c3telemetry@stageazpaasstore01.blob.core.windows.net/")
// def path = sys.env.get("WASB_PATH").getOrElse("wasb://sean-test@stageazpaasstore01.blob.core.windows.net/")

def fs = FileSystem.get(URI.create(path), new Configuration())

var exitAfter = false

def tenants(include: Set[String] = Set.empty) = {
  val bl = Set("c3telemetry", "sean-test", "")
  val Pattern = ".*/(.*)".r

  fs.listStatus(new Path(path)).map(
    _.getPath.toString
  ).map {
    case Pattern(tenant) => tenant
  }.filter { 
    case p if p.startsWith("_") || bl.contains(p) => false
    case p if !include.isEmpty && !include.contains(p) => false
    case _ => true
  }.sorted
}

def partitions(prefix: String, key: String)(implicit dates: (String, String)) = {
  val pattern = ".*/" + key + "=(\\d{" + key.length + "})"
  val LeafValuePattern = pattern.r
  val DatePattern = ".*?/(.*)".r

  fs.listStatus(new Path(s"$path$prefix")).map(
    _.getPath.toString
  ).map {
    case LeafValuePattern(value) => s"$prefix/$key=$value"
    case _ => null
  }.filter{
    /*
     * when the range is (2021-12-28 - 2022-01-03),
     * 2020 is not between (2012 - 2022)
     * 2020-01 to 2021-11 is not between (2021-12 - 2022-01)
     * 2021-12-01 to 2021-12-27 is not between (2021-12-28 to 2022-01-03)
     * and so forth
     */
    case DatePattern(date: String) if 
      date >=  dates._1.substring(0, date.length) && 
      date <= dates._2.substring(0, date.length) => true
    case _ => false
  }
}

def copy(include: Set[String] = Set.empty, to: String = "sean-metry")
        (implicit dates: (String, String) = ("yyyy=2020/mm=01/dd=01", "yyyy=2040/mm=12/dd=31")) = {
  val dest = s"wasb://$to@stageazpaasstore01.blob.core.windows.net/"
  def fs2 = FileSystem.get(URI.create(dest), new Configuration())

  val years = tenants(include).flatMap(prefix => partitions(prefix, "yyyy"))
  spark.sparkContext.parallelize(years).map { f =>
    val destf = f.replace("c3telemetry", to)
    println(s"copy($path$f, $dest$destf)")
    FileUtil.copy(fs, new Path(s"$path$f"), fs2, new Path(s"$dest$destf"), false, new Configuration())
  }.collect
}

def process(dryRun: Boolean = true, include: Set[String] = Set.empty)
           (implicit dates: (String, String) = ("yyyy=2020/mm=01/dd=01", "yyyy=2040/mm=12/dd=31")) = {
  println(s"Processing ${if(dryRun) "DRY" else "NON-DRY"} on $path between ${dates._1} and ${dates._2}...")

  val DROP_PARTITION = "ALTER TABLE c3telemetry.raw DROP IF EXISTS PARTITION"
  val ADD_PARTITION = "ALTER TABLE c3telemetry.raw ADD PARTITION"
  val Pattern = "(.*)/yyyy=(.*)/mm=(.*)/dd=(.*)".r
  val runId = scala.util.Random.alphanumeric.take(8).mkString("")
  val output = s"/tmp/spark/$runId/out"
  val logs = s"/tmp/spark/$runId/log"
  s"mkdir -p $logs".!

  // make tenant + year as a spark partition
  val years = tenants(include).flatMap(prefix => partitions(prefix, "yyyy"))
  val rdd = spark.sparkContext.parallelize(years)

  // generate sql text files per partition: out/part-00001, for example
  rdd.flatMap{ 
    partitions(_, "mm").flatMap(partitions(_, "dd"))
  }.map{ 
    case Pattern(tenant, yyyy, mm, dd) => 
      s"""$DROP_PARTITION (cluster="$tenant", yyyy="$yyyy", mm="$mm", dd="$dd");""" + 
      s"""$ADD_PARTITION (cluster="$tenant", yyyy="$yyyy", mm="$mm", dd="$dd") LOCATION "abfs://c3telemetry@stageazpaasstore01.dfs.core.windows.net/$tenant/yyyy=$yyyy/mm=$mm/dd=$dd";"""
  }.saveAsTextFile(output)

  // parallelize importing of the files into impala
  rdd.mapPartitionsWithIndex { (index, itr) =>
    val file = s"$output/part-" + "%05d".format(index)
    val countCommand = s"wc -l $file"
    val importCommand = s"impala -f $file > $logs 2>&"
    val lines = countCommand.!
    if(dryRun) {
      s"echo [${Thread.currentThread.getId}] $importCommand".!
    } else {
      importCommand.!
    } 
    itr // got to return an iterator of string per mapPartitionsWithIndex signature
  }.collect
}

// limit the date range
def processForDays(days: Int = 5, dryRun: Boolean = true, include: Set[String] = Set.empty) = {
  try {
    val min = java.time.LocalDate.now.minusDays(days - 2)
    val max = min.plusDays(days)

    val datef = java.time.format.DateTimeFormatter.ofPattern("'yyyy'=yyyy'/mm'=MM'/dd'=dd");
    implicit val dates = (min.format(datef), max.format(datef))
    process(dryRun, include)
  } catch {
    case e: Exception => 
      e.printStackTrace
      if(exitAfter) System.exit(1)
  } finally {
    if(exitAfter) System.exit(0)
  }
}

// run daily incremental partition refresh 
sys.env.get("SHELL_EXEC_MODE") match {
  case Some(v) if v == "DAILY" => 
    println("Running daily incremental partition refresh...")
    exitAfter = true
    processForDays(dryRun = sys.env.get("DRY") match {
      case Some(v) if v.equalsIgnoreCase("false") => false
      case _ => true
    })
  case _ => // nothing
}
