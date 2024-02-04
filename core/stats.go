package core

// redis by default has 16 dbs(db0 - db15), we support only 1 db, so all the data by default goes to db0.
// current support only exists for number of keys in Keyspace stat
var keyspaceStats [1]map[string]int64

func updateKeyspaceStats(key string, value int64) {
	if keyspaceStats[0] == nil {
		keyspaceStats[0] = make(map[string]int64, 0)
	}

	keyspaceStats[0][key] = value
}

func incrementKeyspaceStats(key string) {
	if keyspaceStats[0] == nil {
		keyspaceStats[0] = make(map[string]int64, 0)
	}
	keyspaceStats[0][key]++
}

func decrementKeyspaceStats(key string) {
	if keyspaceStats[0] == nil {
		keyspaceStats[0] = make(map[string]int64, 0)
	}
	if keyspaceStats[0][key] > 0 {
		keyspaceStats[0][key]--
	}
}
