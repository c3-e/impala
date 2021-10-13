package app

import (
	"encoding/gob"

        "github.com/quasoft/memstore"
	"github.com/joho/godotenv"
)

var (
	Store *memstore.MemStore
)

func Init() error {
	_ = godotenv.Load()
        // Sean A. Ignore env file

	// Sean A. Switched to MemStore
	// Store = sessions.NewFilesystemStore(os.TempDir(), []byte("secret"))
	Store = memstore.NewMemStore(
		[]byte("authkey123"),
		[]byte("enckey12341234567890123456789012"),
	)
	// Store.MaxLength(math.MaxInt64)
	gob.Register(map[string]interface{}{})
	return nil
}
