package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Password    string
	Devices     interface{}
	PasswordMap map[string]string
}

func (c *Config) readConfig(filename string) (err error) {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err

	}

	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return err
	}

	// Make device map
	c.PasswordMap = make(map[string]string)

	a, _ := c.Devices.([]interface{})
	//fmt.Println(len(a))
	for _, x := range a {
		y := x.(map[interface{}]interface{})
		host := fmt.Sprintf("%v", y["host"])
		pass := fmt.Sprintf("%v", y["pass"])
		c.PasswordMap[host] = pass
	}

	return nil

}

/**
return password for device if specified in devices section. Else returns default password.
*/

func (c *Config) getDevicePassword(hostname string) string {
	// return device passwort, if specified in PasswordMap
	if pass, ok := c.PasswordMap[hostname]; ok {
		return pass
	}

	// return default password
	return c.Password
}

/* Main function for test purposes
func main() {

	c := Config{}
    c.readConfig("pjlink.yaml")

	fmt.Println("Pass", c.getDevicePassword("n24-h13-beamer.mt.uni-ulm.de"))



}
*/
