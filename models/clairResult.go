package models



type ClairResult struct {
	ClairId     int    `orm:"pk;column(clair_id)" json:"clair_id"`
	RepoName    string `orm:"column(repo_name)" json:"repo_name"`
	UserName    string `orm:"column(user_name)" json:"user_name"`
	ProjectName string `orm:"column(project_name)" json:"project_name"`
	HighNum     int    `orm:"column(high_num)" json:"high_num"`
	OtherNum    int    `orm:"column(other_num)" json:"other_num"`
	ClairResult string `orm:"column(clair_result)" json:"clair_result"`
	CVEStatus   int    `orm:"column(CVE_status)" json:"CVE_status"`
	OtherStatus int    `orm:"column(other_status)" json:"other_status"`
}