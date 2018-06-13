Release 1.0

实现功能：
1.用户向Harbor中上传镜像时，可实现自动调用clair对该镜像进行漏洞扫描.

2.若镜像存在问题（有漏洞或cliar不支持该镜像的扫描），则会向用户邮箱发送镜像漏洞邮件，详细给出该镜像的漏洞信息，以便用户可以方便的查看.

3.对镜像漏洞给出可修复的修复建议，方便用户可以完善自己的镜像安全。

4.API方面：
    (1)getVulnerabilitySummary：该API可提供指定镜像的全部漏洞信息，包括该镜像包含的漏洞总数、可修复漏洞数、高危漏洞数、中危漏洞数、低危漏洞数、以及可忽略或不识别漏洞数，同时会详细列出每个漏洞信息。
    (2)getVulnerabilitiesByPackage:该API可将指定镜像的漏洞信息按Package分类汇总，给出每个package中漏洞总数、可修复漏洞数、以及各level的漏洞数，同时会详细每个package的漏洞信息。
    (3)getClairStatistcs：该API可将镜像扫描结果按照user和project进行分类，统计每个user/project下的镜像总数和不安全的镜像数。
    (4)getTags：该API可获取指定镜像的tag_name，以及该镜像的高危漏洞数和其他级别漏洞数。



目前bug：

1.Harbor目前不稳定：常出现不能正常工作的状态，例如：docker push完成后，ui界面与mysql中都没有关于上传镜像的内容，并且没有ERROR日志输出。将Harbor down掉后重新up起新的Harbor的，又恢复正常工作。


自动化脚本改动：

1.harbor.cfg
2.harbor-ui.tar


