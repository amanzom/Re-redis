package config

var Host string = "0.0.0.0"
var Port int = 7369

var NumKeysThresholdForEviction int = 100
var EvictionStrategy = "simple-first"

var AofFilePath = "./re-redis.aof" // run to give read and write access these files - `chmod +w ./re-redis.aof, chmod +r ./re-redis.aof`
var TempAofFilePath = "./temp.aof" // to check aof file is valid or not: redis-check-aof ./re-redis.aof

var StoreReconstructEnabledOnBootUp = true
