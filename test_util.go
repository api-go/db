package db

// For test only

func ResetConfigsForTest() {
	dbConfigsLock.Lock()
	clear(dbConfigs)
	dbConfigsLock.Unlock()
}

func ResetInstancesForTest() {
	dbInstancesLock.Lock()
	for _, db := range dbInstances {
		db.conn.Close()
	}
	clear(dbInstances)
	dbInstancesLock.Unlock()
}

func ResetAllForTest() {
	ResetConfigsForTest()
	ResetInstancesForTest()
}

func SetConfigForTest(name string, conf *Config) {
	dbConfigsLock.Lock()
	dbConfigs[name] = conf
	dbConfigsLock.Unlock()
}
