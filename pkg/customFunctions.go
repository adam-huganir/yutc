package yutc

import "gopkg.in/yaml.v3"

func MustToYaml(v interface{}) (string, error) {
	var err error
	var out []byte
	if out, err = yaml.Marshal(v); err != nil {
		return "", err
	}
	return string(out), nil
}

func ToYaml(v interface{}) string {
	out, _ := MustToYaml(v)
	return out
}

func MustFromYaml(s string) (interface{}, error) {
	var err error
	var out interface{}
	if err = yaml.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	return out, nil
}

func FromYaml(s string) interface{} {
	out, _ := MustFromYaml(s)
	return out
}
