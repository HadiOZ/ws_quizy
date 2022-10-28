package pool

// defaul Pool
var defPool map[string]*Room

func init() {
	defPool = make(map[string]*Room)
	go func() {
		for code, v := range defPool {
			if v.Len() == 0 {
				delete(defPool, code)
			}
		}
	}()
}

func Search(code string) *Room {
	return defPool[code]
}

func Add(code string, room *Room) {
	defPool[code] = room
}

func Check(code string, uniqcode string) (bool, *Room) {
	room, val := defPool[code]
	for _, r := range room.connections {
		if r.Uniqcode() == uniqcode {
			return val, room
		}
	}
	return val, nil
}
