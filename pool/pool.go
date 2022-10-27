package pool

var Pool map[string]*Room

func init() {
	Pool = make(map[string]*Room)
	go func() {
		for code, v := range Pool {
			if v.LenConn() == 0 {
				delete(Pool, code)
			}
		}
	}()
}
