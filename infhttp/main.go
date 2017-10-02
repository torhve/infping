// infhttp.go copyright Tor Hveem
// License: MIT

package main

import (
    "github.com/influxdb/influxdb/client"
    "github.com/pelletier/go-toml"
    "fmt"
    "log"
    "os"
    "net/url"
    "net/http"
    "time"
    "io/ioutil"
)

func herr(err error) {
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func perr(err error) {
    if err != nil {
        fmt.Println(err)
    }
}

func readPoints(config *toml.TomlTree, con *client.Client) {
    urls := config.Get("urls.urls").([]interface{})
    log.Printf("Going to fetch the following urls: %q", urls)
    for {
        for _, v := range urls {
            url, _ := v.(string)
            go func(url string) {
                start := time.Now()
                response, err := http.Get(url)
                perr(err)
                contents, err := ioutil.ReadAll(response.Body)
                defer response.Body.Close()
                perr(err)
                elapsed := time.Since(start).Seconds()
                code := response.StatusCode
                bytes := len(contents)
                log.Printf("Url:%s, code: %s, bytes: %s, elapsed: %s", url, code, bytes, elapsed)
                writePoints(config, con, url, code, bytes, elapsed)
            }(url)
        }
        time.Sleep(time.Second * 30)
    }
}

func writePoints(config *toml.TomlTree, con *client.Client, url string, code int, bytes int, elapsed float64) {
    db := config.Get("influxdb.db").(string)
    pts := make([]client.Point, 1)
    fields := map[string]interface{}{}
    fields = map[string]interface{}{
        "code": code,
        "bytes": bytes,
        "elapsed": elapsed,
    }
    pts[0] = client.Point{
        Measurement: "http",
        Tags: map[string]string{
            "url": url,
        },
        Fields: fields,
        Time: time.Now(),
        Precision: "s",
    }

    bps := client.BatchPoints{
        Points:          pts,
        Database:        db,
        RetentionPolicy: "default",
    }
    _, err := con.Write(bps)
    perr(err)
}

func main() {
    config, err := toml.LoadFile("config.toml")
    if err != nil {
        fmt.Println("Error:", err.Error())
        os.Exit(1)
    }

    host := config.Get("influxdb.host").(string)
    port := config.Get("influxdb.port").(string)
    //measurement := config.Get("influxdb.measurement").(string)
    username := config.Get("influxdb.user").(string)
    password := config.Get("influxdb.pass").(string)

    u, err := url.Parse(fmt.Sprintf("http://%s:%s", host, port))
    if err != nil {
        log.Fatal(err)
    }

    conf := client.Config{
        URL:      *u,
        Username: username,
        Password: password,
    }

    con, err := client.NewClient(conf)
    if err != nil {
        log.Fatal(err)
    }

    dur, ver, err := con.Ping()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Connected to influxdb! %v, %s", dur, ver)

    readPoints(config, con)
}
