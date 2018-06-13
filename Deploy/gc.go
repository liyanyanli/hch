package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/robfig/cron"
)

func main() {
	spec := "0 00 00 * * *"
	c := cron.New()
	c.AddFunc(spec, GarbageCollection)
	c.Start()
	select {}
}

//调用系统指令的方法，参数s 就是调用的shell命令
func GarbageCollection() {
	cmd := exec.Command("/bin/sh", "-c", "docker ps | grep 5000/tcp") //调用Command函数
	out, err := cmd.Output()
	cid := strings.Split(string(out), " ")[0]
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Starting the garbage collection process. \n") //输出执行结果

	cmd = exec.Command("/bin/sh", "-c", "docker stop "+cid) //调用Command函数
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The registry container has stopped. \n") //输出执行结果

	cmd = exec.Command("/bin/sh", "-c", "docker run -i --name gc --rm --volumes-from "+cid+" registry:2.5.0 garbage-collect --dry-run /etc/registry/config.yml") //调用Command函数
	gc, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s \n", gc) //输出执行结果

	cmd = exec.Command("/bin/sh", "-c", "docker start "+cid) //调用Command函数
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The registry container has starting working. \n") //输出执行结果

}
