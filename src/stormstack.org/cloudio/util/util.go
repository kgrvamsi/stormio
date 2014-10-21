package util

import (
	"stormstack.org/cloudio/conf"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/cihub/seelog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	defaultLocation = "/etc/cloudio/default.cfg"
)

var (
	Config      *conf.ConfigFile
	profilePath string = defaultLocation
)

type genericMap map[string]interface{}
type Request genericMap
type Response genericMap

func ToString(val interface{}) (s string) {
	b, err := json.Marshal(val)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}

func ToObject(val []byte, obj interface{}) error {
	err := json.Unmarshal([]byte(val), obj)
	return err
}

func (r Response) String() (s string) {
	b, err := json.Marshal(r)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}

func (r *Request) Load(req *http.Request) error {
	var jr Request = Request(*r)
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&jr)
	*r = Request(jr)
	return err
}

type Filter struct {
	Params url.Values
}

// NewFilter creates a new Filter.
func NewFilter() *Filter {
	return &Filter{make(url.Values)}
}

func (f *Filter) Set(filter, value string) {
	f.Params.Set(filter, value)
}

func LoadProperties() {
        fmt.Printf("First Attempt: reading Profile from the command line...\n")
        flag.StringVar(&profilePath, "profile", "", "You can redefine the custom properties like path of the log etc..")
        flag.Parse()

        // Skip checking profile path if no flags processed
        found := false
        nf := flag.NFlag()
        if nf > 0 {
                // Change the profile name passed to absolute path. This helps in reading from CWD as well
                // Make sure profilePath is not empty
                fmt.Printf("profilepath is %s, length is %d\n", profilePath, len(profilePath))
                if profilePath == "" {
                        found = false
                } else {
                        filename, _ := filepath.Abs(profilePath)
                        fmt.Printf("filename is %s \n", filename)
                        found, _ = exists(filename)
                }
        }

        if !found {
                fmt.Printf("Second Attempt: reading Profile from Environment Variables\n \tCLOUDIOPATH= %s \n\tCLOUDCONFIG= %s\n",
                                os.Getenv("CLOUDIOPATH"), os.Getenv("CLOUDIOCONFIG"))
                fileName := os.Getenv("CLOUDIOCONFIG")

                // read from the environment
                if fileName == "" {
                        profilePath = os.Getenv("CLOUDIOPATH") + "/cloudio.cfg"
                } else {
                        profilePath = os.Getenv("CLOUDIOPATH") + "/" + fileName
                }
                found, _ = exists(profilePath)
        }

        if !found {
                //read from the default location
                fmt.Printf("Third Attempt: reading the profile in the default location(%s)..\n", defaultLocation)
                profilePath = defaultLocation
                found, _ = exists(profilePath)
        }

        if !found {
                fmt.Println("FATAL Error: Profile didn't found any where, terminating the program\n")
                os.Exit(2)
        }
        fmt.Printf("Successfully read Profile from :%s\n", profilePath)
        Config, _ = conf.ReadConfigFile(profilePath)

}

func GetBool(catag, key string) bool {
	val, err := Config.GetBool(catag, key)
	if err != nil {
		log.Errorf("Key %s not found in the [%s]", key, catag)
		return false
	}
	return val
}

func GetInt(catag, key string) int {
	val, err := Config.GetInt(catag, key)
	if err != nil {
		log.Errorf("Key %s not found in the [%s]", key, catag)
		return 10
	}
	return val
}

func GetString(catag, key string) string {
	val, err := Config.GetString(catag, key)
	if err != nil {
		log.Errorf("Key %s not found in the [%s]", key, catag)
		return ""
	}
	return val
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
