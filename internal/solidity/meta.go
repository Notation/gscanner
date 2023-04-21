package solidity

import (
	"encoding/json"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	"gscanner/internal/util"
)

// https://github.com/ethereum/solc-bin/tree/gh-pages/bin
type SolcBinaryVersionInfo struct {
	Path       string   `json:"path"`
	Version    string   `json:"version"`
	Build      string   `json:"build"`
	LogVersion string   `json:"longVersion"`
	Keccak256  string   `json:"keccak256"`
	Sha256     string   `json:"sha256"`
	URLs       []string `json:"urls"`
}

type SolcBinaryMeta struct {
	Builds []SolcBinaryVersionInfo `json:"builds"`
}

func NewSolcBinaryMeta() (*SolcBinaryMeta, error) {
	localMetaFilePath := path.Join(SolcBinaryDir, SolcBinaryMetaFile)
	metaFileExists, err := util.FileExists(localMetaFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "FileExists")
	}
	if !metaFileExists {
		err := util.DownloadFile(localMetaFilePath, SolcBinaryEndpoint+SolcBinaryMetaFile)
		if err != nil {
			return nil, errors.Wrap(err, "DownloadFile")
		}
	}
	return readSolcMeta(localMetaFilePath)
}

// GetSolcBinary 从版本列表中取对应的版本
func (sbm *SolcBinaryMeta) GetSolcBinary(version string) (string, error) {
	version = strings.TrimPrefix(version, "^")
	var solcBinaryPath string
	for i := range sbm.Builds {
		if !strings.Contains(sbm.Builds[i].Path, version) ||
			strings.Contains(sbm.Builds[i].Path, "nightly") {
			continue
		}
		solcBinaryPath = sbm.Builds[i].Path
		break
	}
	if solcBinaryPath == "" {
		return "", errors.Errorf("no version matches")
	}
	localSolcBinaryPath := path.Join(SolcBinaryDir, solcBinaryPath)
	binaryFileExists, err := util.FileExists(localSolcBinaryPath)
	if err != nil {
		return "", errors.Wrap(err, "FileExists")
	}
	if binaryFileExists {
		return localSolcBinaryPath, nil
	}
	err = util.DownloadFile(localSolcBinaryPath, SolcBinaryEndpoint+solcBinaryPath)
	if err != nil {
		return "", errors.Wrap(err, "DownloadFile")
	}
	return localSolcBinaryPath, nil
}

func readSolcMeta(filePath string) (*SolcBinaryMeta, error) {
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "ReadFile")
	}
	var solcMeta SolcBinaryMeta
	err = json.Unmarshal(fileData, &solcMeta)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return &solcMeta, nil
}
