package builtin

import (
	"bytes"
	"encoding/csv"
	builtinJSON "encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/httprunner/httprunner/hrp/internal/json"
)

func Dump2JSON(data interface{}, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		log.Error().Err(err).Msg("convert absolute path failed")
		return err
	}
	log.Info().Str("path", path).Msg("dump data to json")
	file, _ := json.MarshalIndent(data, "", "    ")
	err = os.WriteFile(path, file, 0644)
	if err != nil {
		log.Error().Err(err).Msg("dump json path failed")
		return err
	}
	return nil
}

func Dump2YAML(data interface{}, path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		log.Error().Err(err).Msg("convert absolute path failed")
		return err
	}
	log.Info().Str("path", path).Msg("dump data to yaml")

	// init yaml encoder
	buffer := new(bytes.Buffer)
	encoder := yaml.NewEncoder(buffer)
	encoder.SetIndent(4)

	// encode
	err = encoder.Encode(data)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, buffer.Bytes(), 0644)
	if err != nil {
		log.Error().Err(err).Msg("dump yaml path failed")
		return err
	}
	return nil
}

func FormatResponse(raw interface{}) interface{} {
	formattedResponse := make(map[string]interface{})
	for key, value := range raw.(map[string]interface{}) {
		// convert value to json
		if key == "body" {
			b, _ := json.MarshalIndent(&value, "", "    ")
			value = string(b)
		}
		formattedResponse[key] = value
	}
	return formattedResponse
}

func ExecCommand(cmd *exec.Cmd, cwd string) error {
	log.Info().Str("cmd", cmd.String()).Str("cwd", cwd).Msg("exec command")
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	out := strings.TrimSpace(string(output))
	if err != nil {
		log.Error().Err(err).Str("output", out).Msg("exec command failed")
	} else if len(out) != 0 {
		log.Info().Str("output", out).Msg("exec command success")
	}
	return err
}

func CreateFolder(folderPath string) error {
	log.Info().Str("path", folderPath).Msg("create folder")
	err := os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		log.Error().Err(err).Msg("create folder failed")
		return err
	}
	return nil
}

func CreateFile(filePath string, data string) error {
	log.Info().Str("path", filePath).Msg("create file")
	err := os.WriteFile(filePath, []byte(data), 0o644)
	if err != nil {
		log.Error().Err(err).Msg("create file failed")
		return err
	}
	return nil
}

// IsPathExists returns true if path exists, whether path is file or dir
func IsPathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// IsFilePathExists returns true if path exists and path is file
func IsFilePathExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// path not exists
		return false
	}

	// path exists
	if info.IsDir() {
		// path is dir, not file
		return false
	}
	return true
}

// IsFolderPathExists returns true if path exists and path is folder
func IsFolderPathExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// path not exists
		return false
	}

	// path exists and is dir
	return info.IsDir()
}

func EnsureFolderExists(folderPath string) error {
	if !IsPathExists(folderPath) {
		err := CreateFolder(folderPath)
		return err
	} else if IsFilePathExists(folderPath) {
		return fmt.Errorf("path %v should be directory", folderPath)
	}
	return nil
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetRandomNumber(min, max int) int {
	if min > max {
		return 0
	}
	r := rand.Intn(max - min + 1)
	return min + r
}

func Interface2Float64(i interface{}) (float64, error) {
	switch i.(type) {
	case int:
		return float64(i.(int)), nil
	case int32:
		return float64(i.(int32)), nil
	case int64:
		return float64(i.(int64)), nil
	case float32:
		return float64(i.(float32)), nil
	case float64:
		return i.(float64), nil
	case string:
		intVar, err := strconv.Atoi(i.(string))
		if err != nil {
			return 0, err
		}
		return float64(intVar), err
	}
	// json.Number
	value, ok := i.(builtinJSON.Number)
	if ok {
		return value.Float64()
	}
	return 0, errors.New("failed to convert interface to float64")
}

var ErrUnsupportedFileExt = fmt.Errorf("unsupported file extension")

// LoadFile loads file content with file extension and assigns to structObj
func LoadFile(path string, structObj interface{}) (err error) {
	log.Info().Str("path", path).Msg("load file")
	file, err := readFile(path)
	if err != nil {
		return errors.Wrap(err, "read file failed")
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".json", ".har":
		decoder := json.NewDecoder(bytes.NewReader(file))
		decoder.UseNumber()
		err = decoder.Decode(structObj)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(file, structObj)
	default:
		err = ErrUnsupportedFileExt
	}
	return err
}

func loadFromCSV(path string) []map[string]interface{} {
	log.Info().Str("path", path).Msg("load csv file")
	file, err := readFile(path)
	if err != nil {
		log.Error().Err(err).Msg("read csv file failed")
		panic(err)
	}

	r := csv.NewReader(strings.NewReader(string(file)))
	content, err := r.ReadAll()
	if err != nil {
		log.Error().Err(err).Msg("parse csv file failed")
		panic(err)
	}
	var result []map[string]interface{}
	for i := 1; i < len(content); i++ {
		row := make(map[string]interface{})
		for j := 0; j < len(content[i]); j++ {
			row[content[0][j]] = content[i][j]
		}
		result = append(result, row)
	}
	return result
}

func readFile(path string) ([]byte, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("convert absolute path failed")
		return nil, err
	}

	file, err := os.ReadFile(path)
	if err != nil {
		log.Error().Err(err).Msg("read file failed")
		return nil, err
	}
	return file, nil
}
