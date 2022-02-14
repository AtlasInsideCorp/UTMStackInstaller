package utils

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	sigar "github.com/cloudfoundry/gosigar"
)

func InstallMaster(mode, datadir, pass, tag string, lite bool) error {
	if lite {
		if err := CheckCPU(4); err != nil {
			return err
		}
		if err := CheckMem(3); err != nil {
			return err
		}
	} else {
		if err := CheckCPU(4); err != nil {
			return err
		}
		if err := CheckMem(7); err != nil {
			return err
		}
	}

	esData := MakeDir(0777, datadir, "opendistro", "data")
	esBackups := MakeDir(0777, datadir, "opendistro", "backups")
	cert := MakeDir(0777, datadir, "cert")
	logstashPipeline := MakeDir(0777, datadir, "logstash", "pipeline")
	datasourcesDir := MakeDir(0777, datadir, "datasources")
	rules := MakeDir(0777, datadir, "rules")

	serverName, err := os.Hostname()
	if err != nil {
		return err
	}

	mainIP, err := GetMainIP()
	if err != nil {
		return err
	}

	mainIface, err := GetMainIface(mode)
	if err != nil {
		return err
	}

	m := sigar.Mem{}
	m.Get()
	memory := m.Total / 1024 / 1024 / 1024 / 3

	var updates uint32

	if tag == "testing" {
		updates = 600
	} else {
		updates = 3600
	}

	env := []string{
		"SERVER_TYPE=aio",
		"LITE=" + strconv.FormatBool(lite),
		"SERVER_NAME=" + serverName,
		"DB_HOST=" + mainIP,
		"DB_PASS=" + pass,
		fmt.Sprint("ES_MEM=", memory),
		fmt.Sprint("UPDATES=", updates),
		"ES_DATA=" + esData,
		"ES_BACKUPS=" + esBackups,
		"CERT=" + cert,
		"LOGSTASH_PIPELINE=" + logstashPipeline,
		"UTMSTACK_DATASOURCES=" + datasourcesDir,
		"SCANNER_IFACE=" + mainIface,
		"SCANNER_IP=" + mainIP,
		"CORRELATION_URL=http://" + mainIP + ":9090/v1/newlog",
		"UTMSTACK_RULES=" + rules,
		"TAG=" + tag,
	}

	if !lite {
		if err := InstallScanner(mode); err != nil {
			return err
		}

		if err := InstallSuricata(mode, mainIface); err != nil {
			return err
		}
	}

	// Generate auto-signed cert and key
	if err := generateCerts(cert); err != nil {
		return err
	}

	if err := InitDocker(mode, env, true, tag, lite, mainIP); err != nil {
		return err
	}

	// configure elastic
	if err := initializeElastic(); err != nil {
		return err
	}

	// Initialize PostgreSQL Database
	if err := initializePostgres(pass); err != nil {
		return err
	}

	// Install OpenVPN Master
	if err := InstallOpenVPNMaster(mode); err != nil {
		return err
	}

	baseURL := "https://127.0.0.1/"

	for intent := 0; intent <= 5; intent++ {
		time.Sleep(2 * time.Minute)

		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: transCfg}

		_, err := client.Get(baseURL + "/utmstack/api/ping")

		if err == nil {
			break
		}
	}

	return nil
}
