package api

import (
	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/utils/log"
)

type BenchSecurityAPI struct {
	BaseAPI
}

type AdminOutput struct {
	BenchList []Bench `json:"bench_list"`
}

type Bench struct {
	Ip         string `json:"ip"`
	Host       string `json:"host"`
	Daemon     string `json:"daemon"`
	DaemonFile string `json:"daemonFile"`
	Images     string `json:"images"`
	//Runtime    string `json:"runtime"`
	Operation  string `json:"operation"`
}



type RuntimeIn struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Runtime       string `json:"runtime"`
}

type RuntimeInList struct {
	Ip          string `json:"ip"`
	RunTimeList []RuntimeIn `json:"runTime_list"`
}

type RunTimeSesult struct {
	Runtime string `json:"runtime_result"`
}

type RunTimeResult struct {
	Id      string `json:"Continer_id"`
	Name    string `json:"Continer_name"`
	Runtime string `json:"runtime_result"`
}

type RunTimeResults struct {
	Result []RunTimeResult `json:"results"`
}



func (si *BenchSecurityAPI) SaveBenchSecurity() {
	var input Bench
	si.DecodeJSONReq(&input)
	if input.Ip == "" {
		log.Error("error happens ip is nil")
		return
	}

	count, err := dao.GetIpCount(input.Ip)
	if err != nil {
		log.Errorf("error happens get ip count: %v", err)
		return
	}

	if count > 0 {
		err := dao.UpdateBenchResult(input.Ip,input.Host,input.Daemon,input.DaemonFile,input.Images,input.Operation)
		if err != nil {
			log.Errorf("Error happens update bench_security: %v", err)
		}

	} else {
		//insert into mysql
		pid, err := dao.AddBenchResult(input.Ip,input.Host,input.Daemon,input.DaemonFile,input.Images,input.Operation)
		if err != nil {
			log.Errorf("Error happens insert bench_security: %v", err)
		}
		log.Debugf("The result of insert bench_security: %s", pid)
	}
}

func (ra *BenchSecurityAPI) GetBenchSecurity() {
	ra.ValidateUser()
	var bench Bench
	var benchList  []Bench
	var benches AdminOutput
	benchs ,err := dao.GetAllBench()

	if err != nil {
		log.Errorf("Error happens get all repo_name: %v", err)
	}
	for _, b := range benchs {
		bench.Ip = b.Ip
		bench.Host = b.Host
		bench.Daemon = b.Daemon
		bench.DaemonFile = b.DaemonFile
		bench.Images = b.Images
		//bench.Runtime = b.Runtime
		bench.Operation = b.Operation
		benchList = append(benchList, bench)
		bench = clean()
	}
	benches.BenchList = benchList
	ra.Data["json"] = benches
	ra.ServeJSON()
}

func (si *BenchSecurityAPI) SaveBenchSecurityRuntime() {
	var inputList RuntimeInList
	si.DecodeJSONReq(&inputList)

	if inputList.Ip == "" || len(inputList.RunTimeList) == 0 {
		log.Error("error happens ip or runtime result is nil")
		return
	}

	ip := inputList.Ip

	for _, inputRuntime := range inputList.RunTimeList {
		count, err := dao.GetContainerIDCount(inputRuntime.ContainerID)
		if err != nil {
			log.Errorf("error happens get ContainerID count: %v", err)
			return
		}
		if count > 0 {
			err := dao.UpdateRuntime(inputRuntime.ContainerID,inputRuntime.Runtime)
			if err != nil {
				log.Errorf("Error happens update runtime: %v", err)
			}

		} else {
			//insert into mysql
			pid, err := dao.AddRuntime(inputRuntime.ContainerID,inputRuntime.ContainerName,inputRuntime.Runtime,ip)
			if err != nil {
				log.Errorf("Error happens insert runtime: %v", err)
			}
			log.Debugf("The result of insert runtime: %s", pid)
		}

	}
}


func (ra *BenchSecurityAPI) GetBenchSecurityRuntime() {
	ra.ValidateUser()
	id := ra.GetString("container_id")
	var runtimeR RunTimeSesult
	runtime ,err := dao.GetRunTimeById(id)
	if err != nil {
		log.Errorf("Error happens get BenchSecurityRuntime by id: %v", err)
	}
	runtimeR.Runtime = runtime
	ra.Data["json"] = runtimeR
	ra.ServeJSON()
}

func clean() (b Bench) {
	return Bench{}
}

func (ra *BenchSecurityAPI) GetRuntimeByIp() {
	ra.ValidateUser()
	ip := ra.GetString("ip")
	var runtimeR RunTimeResult
	var rs RunTimeResults
	runtimes ,err := dao.GetRunTimeByIp(ip)
	if err != nil {
		log.Errorf("Error happens get BenchSecurityRuntime by ip: %v", err)
	}

	for _, runtime := range runtimes {
		runtimeR.Id = runtime.Continer_id
		runtimeR.Name = runtime.Continer_name
		runtimeR.Runtime = runtime.Container_runtime
		rs.Result = append(rs.Result,runtimeR)
	}

	ra.Data["json"] = rs
	ra.ServeJSON()
}

