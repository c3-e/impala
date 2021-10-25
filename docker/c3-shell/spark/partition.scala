import java.net.URI
import org.apache.hadoop.fs._
import org.apache.hadoop.conf.Configuration
import sys.process._

def path = "wasb://c3telemetry@stageazpaasstore01.blob.core.windows.net/"
// def path = "wasb://sean-test@stageazpaasstore01.blob.core.windows.net/"
def fs = FileSystem.get(URI.create(path), new Configuration())

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

def partitions(prefix: String, key: String) = {
  val pattern = ".*/" + key + "=(\\d{" + key.length + "})"
  val Pattern = pattern.r

  fs.listStatus(new Path(s"$path$prefix")).map(
    _.getPath.toString
  ).map {
    case Pattern(value) => s"$prefix/$key=$value"
    case _ => null
  }.filter{
    case partition if partition != null => true
    case _ => false
  }
}

def copy(include: Set[String] = Set.empty, to: String = "sean-metry") = {
  val dest = s"wasb://$to@stageazpaasstore01.blob.core.windows.net/"
  def fs2 = FileSystem.get(URI.create(dest), new Configuration())

  val years = tenants(include).flatMap(prefix => partitions(prefix, "yyyy"))
  spark.sparkContext.parallelize(years).map { f =>
    val destf = f.replace("c3telemetry", to)
    println(s"copy($path$f, $dest$destf)")
    FileUtil.copy(fs, new Path(s"$path$f"), fs2, new Path(s"$dest$destf"), false, new Configuration())
  }.collect
}

def process(dryRun: Boolean = true, include: Set[String] = Set.empty) = {
  val DROP_PARTITION = "ALTER TABLE c3telemetry.raw DROP IF EXISTS PARTITION"
  val ADD_PARTITION = "ALTER TABLE c3telemetry.raw ADD PARTITION"
  val Pattern = "(.*)/yyyy=(.*)/mm=(.*)/dd=(.*)".r
  val output = "/tmp/out/" + scala.util.Random.alphanumeric.take(8).mkString("")
  val logs = "/tmp/log/" + scala.util.Random.alphanumeric.take(8).mkString("")
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
    val command = s"impala -f $output/part-" + "%05d".format(index) + s" > $logs 2>&"
    if(dryRun) {
      s"echo [${Thread.currentThread.getId}] $command".!
    } else {
      command.!
    } 
    itr // got to return an iterator of string per mapPartitionsWithIndex signature
  }.collect
}
