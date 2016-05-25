// Author: XJP09_HK <jianping_xie@aliyun.com>

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	Name        string
	Description string
	Url         string
	Regexp      string
	Period      int64
	Email       []string
	Mobile      []string
}

type SmtpConfig struct {
	Server   string
	Port     int64
	User     string
	Password string
}

var smtpConfig = SmtpConfig{
	Server:   "smtp.server.com",
	Port:     25,
	User:     "user@mail.server.com",
	Password: "password",
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func logHelper(level string, v ...interface{}) {
	newValue := make([]interface{}, len(v)+1)
	newValue[0] = level
	copy(newValue[1:], v)
	log.Println(newValue...)
}
func logInfo(v ...interface{}) {
	logHelper("[INFO]", v...)
}

func logWarning(v ...interface{}) {
	logHelper("[WARNING]", v...)
}

func logErr(v ...interface{}) {
	logHelper("[ERROR]", v...)
	runtime.Goexit()
}

func a2f(s string) float64 {
	if strings.Trim(s, " ") == "" {
		return 0
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Println(err)
		f = 0
	}
	return f
}

func f2a(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func sendMail(price float64, t Task) {
	if len(t.Email) <= 0 {
		return
	}

	auth := smtp.PlainAuth("", smtpConfig.User,
		smtpConfig.Password, smtpConfig.Server)

	to := t.Email
	msg := []byte("To: " + strings.Join(t.Email, ",") + "\r\n" +
		"Subject: 预订价格更新啦~\r\n" +
		"\r\n" +
		"你预订的商品地址 " + t.Url + " 的最新价格为 " + f2a(price) + " 元\r\n")
	err := smtp.SendMail(smtpConfig.Server+":"+string(smtpConfig.Port),
		auth, smtpConfig.User, to, msg)
	if err != nil {
		logWarning("send mail error, ", err, t)
		return
	}
	logInfo("sendmail, ", t)
}

func sendMsg(price float64, t Task) {

}

func handleTask(t Task) {
	// 检测相关配置
	if strings.Trim(t.Name, " ") == "" {
		logErr("name of task is empty: ", t)
	}
	url := strings.Trim(t.Url, " ")
	if url == "" {
		logErr("url of task is empty: ", t)
	}
	regText := strings.Trim(t.Regexp, " ")
	if regText == "" {
		logErr("regexp of task is empty: ", t)
	}
	if t.Period <= 0 {
		logErr("invalid task", t)
	}

	logInfo("start task now: ", t)

	// 依据定时配置周期执行
	for _ = range time.Tick(time.Duration(t.Period) * time.Minute) {
		// 抓取 url 内容
		res, err := http.Get(url)
		if err != nil {
			logWarning("fetch url error: ", t, err)
			continue
		}

		if res.StatusCode != 200 {
			logWarning("fetch url error: ", t, res)
			continue
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			logWarning("fetch url error: ", t, err)
			continue
		}

		// 匹配出价格
		reg := regexp.MustCompile(t.Regexp)
		submatch := reg.FindSubmatch(content)
		if submatch == nil {
			logWarning("get price error: ", t, string(content))
			continue
		}
		price := a2f(string(submatch[1]))
		if price == 0 {
			logWarning("empty price: ", t)
			continue
		}
		// 对比最优价格
		pricePath := "./logs/price/"
		err = os.MkdirAll(pricePath, 0755)
		checkErr(err)

		priceFile := pricePath + t.Name
		oldPriceData, _ := ioutil.ReadFile(priceFile)
		oldPrice := a2f(string(oldPriceData))
		if oldPrice <= 0 {
			oldPrice = 100000000
		}
		if price < oldPrice {
			err = ioutil.WriteFile(priceFile, []byte(f2a(price)), 0644)
			if err != nil {
				logErr(err, t)
			}

			// 发送通知
			sendMail(price, t)
			// sendMsg(price, t)
		}
	}
}

func main() {
	// log.SetFlags(log.Lshortfile | log.LstdFlags)

	// 获取任务配置
	taskJson, err := ioutil.ReadFile("./conf/task.json")
	checkErr(err)

	var taskList []Task
	err = json.Unmarshal(taskJson, &taskList)
	checkErr(err)

	if len(taskList) <= 0 {
		logWarning("No task!")
		return
	}

	// 启动任务
	for _, task := range taskList {
		go handleTask(task)
	}
	select {}
}
