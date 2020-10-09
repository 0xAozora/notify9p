package main

import (
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/emirpasic/gods/trees/redblacktree"
	"github.com/rjeczalik/notify"
)

func main() {

	//Read Env
	env := os.Getenv("Watch")
	if env == "" {
		env = "C:\\Users\\Thanos\\go\\src\\notify9p\\watch"
		//log.Fatalln("Environment variable not set ...\n... quitting.")
	}

	//Scan Dir
	dir := redblacktree.NewWithStringComparator()

	err := filepath.Walk(env, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		dir.Put("."+path[len(env):], struct{}{})
		return nil
	})
	if err != nil {
		log.Println(err)
	}

	//Watch Dir
	c := make(chan notify.EventInfo, 1)

	if err := notify.Watch(env+"/...", c, notify.Create, notify.Remove, notify.Rename); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	//Socket
	ln, err := net.Listen("tcp", ":5640")
	if err != nil {
		log.Fatalln("Failed opening socket at port 5640")
	}
	l := make(chan net.Conn, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err == nil {
				l <- conn
			}
		}
	}()

	//Wait for events
	newpath := ""
	for {
		select {
		case conn := <-l:
			println("Hello Con")
			s := ""
			for _, path := range dir.Keys() {
				s += "\n" + path.(string)
			}
			conn.Write([]byte(s[1:]))
			conn.Close()
		case e := <-c:
			path := "." + e.Path()[len(env):]
			switch e.Event() {
			case notify.Create:
				dir.Put(path, struct{}{})
			case notify.Remove:
				dir.Remove(path)
				path += "/"
				length := len(path)
				for {
					n, found := dir.Ceiling(path)
					if !(found && len(n.Key.(string)) >= length && n.Key.(string)[:length] == path) {
						break
					}
					dir.Remove(n.Key)
				}
			case notify.Rename:
				if newpath == "" {
					newpath = path
					dir.Put(path, struct{}{})
				} else {
					dir.Remove(path)
					path += "/"
					length := len(path)
					for {
						n, found := dir.Ceiling(path)
						if !(found && len(n.Key.(string)) >= length && n.Key.(string)[:length] == path) {
							break
						}
						dir.Put(newpath+n.Key.(string)[length-1:], struct{}{})
						dir.Remove(n.Key)
					}
					newpath = ""
				}
			}
		}
	}
}
