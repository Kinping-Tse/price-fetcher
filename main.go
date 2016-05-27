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
	Email       EmailConfig
	Mobile      []string
}

type SmtpConfig struct {
	Server   string
	Port     int64
	User     string
	Password string
}

type EmailConfig struct {
	Recipients []string
	Subject    string
	Content    string
}

type Config struct {
	Smtp SmtpConfig
}

var config Config

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

func i2a(i int64) string {
	return strconv.FormatInt(i, 10)
}

func sendMail(price float64, t Task) {
	if len(t.Email.Recipients) <= 0 {
		return
	}

	r := strings.NewReplacer("{name}", t.Name, "{url}", t.Url, "{price}", f2a(price))
	subject := r.Replace(t.Email.Subject)
	content := r.Replace(t.Email.Content)

	smtpConfig := config.Smtp
	auth := smtp.PlainAuth("", smtpConfig.User,
		smtpConfig.Password, smtpConfig.Server)
	msg := []byte("To: " + strings.Join(t.Email.Recipients, ",") + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" + content + "\r\n")

	err := smtp.SendMail(smtpConfig.Server+":"+i2a(smtpConfig.Port),
		auth, smtpConfig.User, t.Email.Recipients, msg)
	if err != nil {
		logWarning("send mail error, ", err, t)
		return
	}
	logInfo("send mail success, ", t)
}

func sendMsg(price float64, t Task) {

}

func handleTask(t Task) {
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

	// Handle task period
	for _ = range time.Tick(time.Duration(t.Period) * time.Minute) {
		// Get html content
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

		// Find the new price
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

		// Compare the price
		pricePath := "./logs/price/"
		err = os.MkdirAll(pricePath, 0755)
		checkErr(err)

		priceFile := pricePath + t.Name
		oldPriceData, _ := ioutil.ReadFile(priceFile)
		oldPrice := a2f(string(oldPriceData))
		if oldPrice <= 0 {
			oldPrice = 100000000
		}

		// If we get the better price, log it and send notify
		if price < oldPrice {
			err = ioutil.WriteFile(priceFile, []byte(f2a(price)), 0644)
			if err != nil {
				logErr(err, t)
			}

			sendMail(price, t)
			// sendMsg(price, t)
		}
	}
}

func main() {
	// log.SetFlags(log.Lshortfile | log.LstdFlags)

	configJson, err := ioutil.ReadFile("./conf/conf.json")
	checkErr(err)

	err = json.Unmarshal(configJson, &config)
	checkErr(err)

	taskJson, err := ioutil.ReadFile("./conf/task.json")
	checkErr(err)

	var taskList []Task
	err = json.Unmarshal(taskJson, &taskList)
	checkErr(err)

	if len(taskList) <= 0 {
		logWarning("No task!")
		return
	}

	// Start task
	for _, task := range taskList {
		go handleTask(task)
	}

	select {}
}
