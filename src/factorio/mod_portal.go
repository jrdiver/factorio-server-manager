package factorio

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ModPortalStruct struct {
	DownloadsCount int    `json:"downloads_count"`
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	Releases       []struct {
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		InfoJSON    struct {
			FactorioVersion Version `json:"factorio_version"`
		} `json:"info_json"`
		ReleasedAt    time.Time `json:"released_at"`
		Sha1          string    `json:"sha1"`
		Version       Version   `json:"version"`
		Compatibility bool      `json:"compatibility"`
	} `json:"releases"`
	Summary string `json:"summary"`
	Title   string `json:"title"`
}

// get all mods uploaded to the factorio modPortal
func ModPortalList() (interface{}, error, int) {
	req, err := http.NewRequest(http.MethodGet, "https://mods.factorio.com/api/mods?page_size=max", nil)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(text)), resp.StatusCode
	}

	var jsonVal interface{}
	err = json.Unmarshal(text, &jsonVal)
	if err != nil {
		return "error", err, http.StatusInternalServerError
	}

	return jsonVal, nil, resp.StatusCode
}

// get the details (mod-info, releases, etc.) from a specific mod from the modPortal
func ModPortalModDetails(modId string) (ModPortalStruct, error, int) {
	var mod ModPortalStruct

	req, err := http.NewRequest(http.MethodGet, "https://mods.factorio.com/api/mods/"+modId, nil)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	err = json.Unmarshal(text, &mod)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return ModPortalStruct{}, errors.New(string(text)), resp.StatusCode
	}

	server := GetFactorioServer()

	installedBaseVersion := Version{}
	_ = installedBaseVersion.UnmarshalText([]byte(server.BaseModVersion))
	requiredVersion := NilVersion

	for key, release := range mod.Releases {
		requiredVersion = release.InfoJSON.FactorioVersion
		release.Compatibility = installedBaseVersion.Compatible(requiredVersion, ">=")
		mod.Releases[key] = release
	}

	return mod, nil, resp.StatusCode
}

// Log the user into factorio, so mods can be downloaded
func FactorioLogin(username string, password string) (error, int) {
	var err error

	resp, err := http.PostForm("https://auth.factorio.com/api-login",
		url.Values{"require_game_ownership": {"true"}, "username": {username}, "password": {password}})

	if err != nil {
		return err, http.StatusInternalServerError
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	bodyString := string(bodyBytes)

	if resp.StatusCode != http.StatusOK {
		return errors.New(bodyString), resp.StatusCode
	}

	var successResponse []string
	err = json.Unmarshal(bodyBytes, &successResponse)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	credentials := Credentials{
		Username: username,
		Userkey:  successResponse[0],
	}

	err = credentials.Save()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	return nil, http.StatusOK
}

// ModPortalFullRelease is a mod release record that includes dependency information.
type ModPortalFullRelease struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	InfoJSON    struct {
		FactorioVersion Version  `json:"factorio_version"`
		Dependencies    []string `json:"dependencies"`
	} `json:"info_json"`
	ReleasedAt    time.Time `json:"released_at"`
	Sha1          string    `json:"sha1"`
	Version       Version   `json:"version"`
	Compatibility bool      `json:"compatibility"`
}

type ModPortalFullStruct struct {
	Name     string                 `json:"name"`
	Title    string                 `json:"title"`
	Releases []ModPortalFullRelease `json:"releases"`
}

// ModPortalModDetailsFull returns mod info including per-release dependency lists
// by calling the /full endpoint of the Factorio mod portal.
func ModPortalModDetailsFull(modId string) (ModPortalFullStruct, error, int) {
	var mod ModPortalFullStruct

	req, err := http.NewRequest(http.MethodGet, "https://mods.factorio.com/api/mods/"+modId+"/full", nil)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()

	text, err := io.ReadAll(resp.Body)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	if resp.StatusCode != http.StatusOK {
		return ModPortalFullStruct{}, errors.New(string(text)), resp.StatusCode
	}

	err = json.Unmarshal(text, &mod)
	if err != nil {
		return mod, err, http.StatusInternalServerError
	}

	server := GetFactorioServer()
	installedBaseVersion := Version{}
	_ = installedBaseVersion.UnmarshalText([]byte(server.BaseModVersion))
	for key, release := range mod.Releases {
		release.Compatibility = installedBaseVersion.Compatible(release.InfoJSON.FactorioVersion, ">=")
		mod.Releases[key] = release
	}

	return mod, nil, resp.StatusCode
}

// parseDependencyName parses a Factorio dependency string and returns the mod name.
// Returns ("", false) for optional, incompatible, or no-effect dependencies.
func parseDependencyName(dep string) (string, bool) {
	dep = strings.TrimSpace(dep)
	if strings.HasPrefix(dep, "(?)") || strings.HasPrefix(dep, "?") ||
		strings.HasPrefix(dep, "!") || strings.HasPrefix(dep, "~") {
		return "", false
	}
	parts := strings.Fields(dep)
	if len(parts) == 0 {
		return "", false
	}
	return parts[0], true
}
