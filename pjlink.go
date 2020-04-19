package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const port = 4352 // The PJLink default TCP - Port

var metrics = []string{"%%1INPT ?", "%%1AVMT ?", "%%1ERST ?", "%%1LAMP ?"} // Send all except power. Power is already send while authentification procedure.

/*
func main() {
	walkpjlink(dest, pass)
}
*/
func walkpjlink(dest string, pass string, pjSlice *[]prometheus.Metric, logger log.Logger) {

	var authenticated = false // Switch for authentication

	level.Debug(logger).Log("msg", "Start PJLink connection to"+dest)
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", dest+":"+strconv.Itoa(port))
	// timeoutDuration := 2 * time.Second
	// conn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		level.Info(logger).Log("msg", "Error scraping target. Connection to device failed", "err", err)
		// Append Metric to result set pjSlice, device is not up
		*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
			prometheus.NewDesc("pjlink_up", "device is up", nil, nil),
			prometheus.GaugeValue,
			float64(0)))
		return
	}
	defer conn.Close() // Als Letztes die Verbindung trennen

	// Alle Nachrichten empfangen, um eine Authentifizierung druchzuführen
	for {
		message, err := bufio.NewReader(conn).ReadString('\r')
		if err != nil {
			fmt.Println(err)
			level.Info(logger).Log("msg", "Error scraping target. Can't read response from device", "err", err)
			return
		}
		level.Debug(logger).Log("msg", "Scraping target. Response from device", "message", message)

		if responseWorker(message, conn, pass, &authenticated, pjSlice, logger) == false {
			break
		}
		if authenticated {
			break
		}
	}

	// Da die Authentifizierung nun stattgefunden hat, können alle Metriken abgefragt werden.
	if authenticated {

		for key, val := range metrics {

			_ = key                     // key declared but not used
			fmt.Fprintf(conn, val+"\r") // send request to device

			message, _ := bufio.NewReader(conn).ReadString('\r') // read response
			level.Debug(logger).Log("msg", "Scraping target. Response from device", "message", message)
			responseWorker(message, conn, pass, &authenticated, pjSlice, logger) // evaluate response
		}

		// Append Metric to result set pjSlice, device is up
		*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
			prometheus.NewDesc("pjlink_up", "device is up", nil, nil),
			prometheus.GaugeValue,
			float64(1)))

	}
}

/*

responseWorker evaluates the devices response

*/

func responseWorker(res string, conn net.Conn, pass string, authenticated *bool, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	a, _ := regexp.Compile("PJLINK 1 [A-Za-z0-9]{8}") // PJLink requires authentification
	b, _ := regexp.Compile("PJLINK ERRA")             // PJLink authentification error
	c, _ := regexp.Compile("PJLINK 0")                // PJLink does not require authentification
	d, _ := regexp.Compile(`\W1POWR\W`)               // Power Response
	e, _ := regexp.Compile(`\W1INPT\W`)               // Input Response
	f, _ := regexp.Compile(`\W1AVMT\W`)               // AV-Mute Response
	g, _ := regexp.Compile(`\W1ERST\W`)               // Error status Response
	h, _ := regexp.Compile(`\W1LAMP\W`)               // Lamp status Response

	if a.MatchString(res) { // device requests an authentification
		level.Info(logger).Log("msg", "Try to authenticate...")
		return authenticate(res, conn, pass, logger)

	} else if b.MatchString(res) { // device sends authentification error
		level.Info(logger).Log("msg", "Error scraping target. PJ Link authentification error")
		return false
	} else if c.MatchString(res) { // device does not require authentification
		level.Debug(logger).Log("msg", "Scraping target. PJ Link does not require authentification")
		fmt.Fprintf(conn, "%%1POWR ?\r")
		return true
	} else if d.MatchString(res) { // update power value
		*authenticated = true // this is the first successfull response after authentification.
		level.Info(logger).Log("msg", "Scraping target. PJ Link authentification successfull")
		return updatePower(res, pjSlice, logger)

	} else if e.MatchString(res) { // update input value
		return updateInput(res, pjSlice, logger)

	} else if f.MatchString(res) { // update AV-Mute value
		return updateAVmute(res, pjSlice, logger)

	} else if g.MatchString(res) { // update error status value
		return updateErst(res, pjSlice, logger)

	} else if h.MatchString(res) { // update lamp status
		return updateLamps(res, pjSlice, logger)

	} else {
		return false
	}

	// return false
}

func authenticate(res string, conn net.Conn, pass string, logger log.Logger) bool {

	// extract the auth token from the response
	runes := []rune(res)
	token := string(runes[9:17]) // it always byte 9 to 17.

	/* Test example with values from the PJLink sheet
	token = "498e4a67"
	pass = "JBMIAProjectorLink"
	*/
	level.Debug(logger).Log("msg", "PJLink Authenticate received token", "token", token)

	hasher := md5.New()
	hasher.Write([]byte(token + pass))

	digest := hex.EncodeToString(hasher.Sum(nil))

	level.Debug(logger).Log("msg", "PJLink Authenticate with String ", "string", digest+"%%1POWR ?")

	fmt.Fprintf(conn, digest+"%%1POWR ?\r")

	return true
}

func extractValues(res string) string {
	// extract the value string from the response
	runes := []rune(res)
	return strings.TrimSpace(string(runes[7:])) // it always from byte 7
}

func updateValue(res string, val string) bool {

	value := extractValues(res)

	fmt.Println(val + " " + value)

	return true

}

func updatePower(res string, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	value := extractValues(res)
	i, _ := strconv.Atoi(value)

	// Append Metric to result set pjSlice
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_power_status", "power status query", nil, nil),
		prometheus.GaugeValue,
		float64(i)))

	level.Debug(logger).Log("msg", "PJLink received power state", "power", i)

	return true

}

func updateInput(res string, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	// Byte 7: 1=RGB, 2=VIDEO, 3= Digital, 4= Storage, 5= Network
	// Byte 8: 1-9
	runes := []rune(res)

	inputClass, _ := strconv.Atoi(string(runes[7]))
	inputNumber, _ := strconv.Atoi(string(runes[8]))

	// Append Metric to result set pjSlice
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_input", "input switch", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(inputClass), "inputClass"))

	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_input", "input switch", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(inputNumber), "inputNumber"))

	level.Debug(logger).Log("msg", "PJLink received input state", "Input Class:", inputClass, "Input Number:", inputNumber)

	return true

}

func updateAVmute(res string, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	// Byte 7: 1=RGB, 2=VIDEO, 3= Digital, 4= Storage, 5= Network
	// Byte 8: 1-9
	value, err := strconv.Atoi(extractValues(res))

	if err != nil {
		println("invalid value", extractValues(res))
	}

	videoMute := 0
	audioMute := 0

	switch value {
	case 11:
		videoMute = 1
		audioMute = 0
	case 21:
		videoMute = 0
		audioMute = 1
	case 31:
		videoMute = 1
		audioMute = 1
	case 30:
		videoMute = 0
		audioMute = 0
	}

	// Append Metric to result set pjSlice
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_av_mute", "audio & video mute status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(videoMute), "video"))

	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_av_mute", "audio & video mute status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(audioMute), "audio"))

	level.Debug(logger).Log("msg", "PJLink received avmute state", " V: ", videoMute, " A: ", audioMute)

	return true

}

func updateErst(res string, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	// Byte 7: FAN error
	// Byte 8: Lamp error
	// Byte 9: temp error
	// Byte 10: open cover
	// Byte 11: Filter error
	// Byte 12: other errors
	// 0 = No Error, 1 = Warning, 2 = error

	runes := []rune(res)

	errFan, _ := strconv.Atoi(string(runes[7]))
	errLamp, _ := strconv.Atoi(string(runes[8]))
	errTemp, _ := strconv.Atoi(string(runes[9]))
	errCover, _ := strconv.Atoi(string(runes[10]))
	errFilter, _ := strconv.Atoi(string(runes[11]))
	errOther, _ := strconv.Atoi(string(runes[12]))

	// Append Metric to result set pjSlice
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errFan), "Fan"))
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errLamp), "Lamp"))

	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errTemp), "Temp"))

	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errCover), "Cover"))
	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errFilter), "Filter"))

	*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
		prometheus.NewDesc("pjlink_error_status", "device error status", []string{"type"}, nil),
		prometheus.GaugeValue,
		float64(errOther), "Other"))

	level.Debug(logger).Log("msg", "PJLink received err state state", "Fan", errFan, "Lamp", errLamp, "Temp", errTemp, "Cover", errCover, "Filter", errFilter, "Other", errOther)

	return true

}

func updateLamps(res string, pjSlice *[]prometheus.Metric, logger log.Logger) bool {

	value := extractValues(res)

	fields := strings.Fields(value)

	for i := 0; i*2 < len(fields); i++ {
		level.Debug(logger).Log("msg", "PJLink received lamp state", "L: ", i+1, "H: ", fields[i*2], "P: ", fields[i*2+1])
		// Append Metric to result set pjSlice
		hours, _ := strconv.Atoi(fields[i*2])
		power, _ := strconv.Atoi(fields[i*2+1])

		*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
			prometheus.NewDesc("pjlink_lamp_hours", "lamp hours", []string{"lamp"}, nil),
			prometheus.GaugeValue,
			float64(hours), strconv.Itoa(i+1)))

		*pjSlice = append(*pjSlice, prometheus.MustNewConstMetric(
			prometheus.NewDesc("pjlink_lamp_power", "lamp power", []string{"lamp"}, nil),
			prometheus.GaugeValue,
			float64(power), strconv.Itoa(i+1)))

	}

	return true

}
