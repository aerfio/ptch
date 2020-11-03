package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/aerfio/ptch/docker"
	"github.com/aerfio/ptch/image"
)

type config struct {
	// read from config file
	Group       string
	ApiEndpoint string
	Token       string
	// flags
	Image  string
	Remote bool
	Help   bool
}

func setupConfig() (config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/ptch")
	viper.AddConfigPath("$HOME/.ptch")
	pflag.BoolP("remote", "r", false, "use image from remote registry like GCR, if not set uses local image")
	pflag.StringP("image", "i", "", "name of image in repo:tag format")
	pflag.BoolP("help", "h", false, "Prints help message")
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return config{}, err
	}
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return config{}, errors.Wrap(err, "configuration file config.yaml not found")
		} else {
			return config{}, errors.Wrap(err, "reading configuration")
		}
	}

	var c config
	if err := viper.Unmarshal(&c); err != nil {
		return config{}, err
	}

	return c, nil
}

func main() {
	config, err := setupConfig()
	if err != nil {
		log.Fatal(err)
	}

	if config.Help {
		pflag.Usage()
		fmt.Println(`In order to use ptch create config.yaml with keys: "group", "token" and "apiEndpoint".`)
		os.Exit(0)
	}

	if config.ApiEndpoint == "" {
		log.Fatal("apiEndpoint fields of config.yaml is not set")
	}

	if config.Image == "" {
		log.Fatal("image is not provided, set it using -i flag")
	}

	if config.Remote {
		reportUrl, err := orderScan(config.Image, config)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(reportUrl)
		os.Exit(0)
	}

	installed := docker.EnsureInstalled()
	if !installed {
		log.Panic("Install docker cli before proceeding")
	}
	running := docker.EnsureRunning()
	if !running {
		log.Panic("Start docker before proceeding")
	}

	osArgImage, err := image.New(config.Image)
	if err != nil {
		log.Panic(err)
	}

	savedImagePath, err := osArgImage.SaveToTmpDir()
	// log.Panic is used everywhere to allow defered func here
	defer func() {
		if _, err := os.Stat(savedImagePath); !os.IsNotExist(err) {
			err := os.Remove(savedImagePath)
			if err != nil {
				log.Println(errors.Wrapf(err, "while deleting tmp docker image: ", savedImagePath))
			}
		}
	}()

	if err != nil {
		log.Panic(err)
	}

	buf, err := ioutil.ReadFile(savedImagePath)
	if err != nil {
		log.Panic(err)
	}

	breader := bytes.NewReader(buf)

	reportUrl, err := uploadDockerImage(breader, osArgImage.String(), config)
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(reportUrl)
}

type Response struct {
	Meta struct {
		Code int `json:"code"`
	} `json:"meta"`
	Results struct {
		Status         string      `json:"status"`
		Sha1Sum        interface{} `json:"sha1sum"`
		ID             int         `json:"id"`
		ProductID      int         `json:"product_id"`
		ReportURL      string      `json:"report_url"`
		Filename       string      `json:"filename"`
		RescanPossible bool        `json:"rescan-possible"`
		Stale          bool        `json:"stale"`
		CustomData     struct {
		} `json:"custom_data"`
		User        string `json:"user"`
		Notify      bool   `json:"notify"`
		LastUpdated string `json:"last_updated"`
	} `json:"results"`
}

func orderScan(img string, c config) (string, error) {
	u, err := url.Parse(c.ApiEndpoint)
	if err != nil {
		return "", errors.Wrap(err, "while parsing apiEndpoint url")
	}
	u.Path = path.Join(u.Path, "api/fetch/")

	req, err := http.NewRequest(http.MethodPost, u.String()+"/" /* path.Join removes that one last / */, nil)
	if err != nil {
		return "", errors.Wrap(err, "while creating request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Group", c.Group)
	req.Header.Set("Url", fmt.Sprintf("docker-registry-https://%s", img))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "while conducting request")
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "while reading response body")
	}
	var report Response
	err = json.Unmarshal(bodyBytes, &report)
	if err != nil {
		return "", errors.Wrapf(err, "while unmarshalling to struct, dumping error as string: %s", string(bodyBytes))
	}

	return report.Results.ReportURL, nil
}

func formatImageName(s string) string {
	name := strings.ReplaceAll(s, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	return strings.ReplaceAll(name, ":", "_")
}

func uploadDockerImage(buf *bytes.Reader, rawName string, c config) (string, error) {
	u, err := url.Parse(c.ApiEndpoint)
	if err != nil {
		return "", errors.Wrap(err, "while parsing apiEndpoint url")
	}
	u.Path = path.Join(u.Path, "api/upload", formatImageName(rawName))
	req, err := http.NewRequest(http.MethodPut, u.String()+"/" /* path.Join removes that one last / */, buf)
	if err != nil {
		return "", errors.Wrap(err, "while creating request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Group", c.Group)
	req.Header.Set("Force-Scan", "True")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "while performing request")
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "while reading response body")
	}

	var report Response
	err = json.Unmarshal(bodyBytes, &report)
	if err != nil {
		return "", errors.Wrap(err, "while unmarshalling to struct")
	}

	return report.Results.ReportURL, nil
}
