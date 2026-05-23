package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"github.com/gorilla/mux"
)

func ModPortalListModsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var statusCode int
	resp, err, statusCode = factorio.ModPortalList()
	w.WriteHeader(statusCode)
	if err != nil {
		resp = fmt.Sprintf("Error in listing mods from mod portal: %s\nresponse: %+v", err, resp)
		log.Println(resp)
		return
	}
}

// ModPortalModInfoHandler returns JSON response with the mod details
func ModPortalModInfoHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	vars := mux.Vars(r)
	modId := vars["mod"]

	var statusCode int
	resp, err, statusCode = factorio.ModPortalModDetails(modId)

	if err != nil {
		resp = fmt.Sprintf("Error in getting mod details from mod portal: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
}

func ModPortalInstallHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	// Get Data out of the request
	var data struct {
		DownloadURL string `json:"downloadUrl"`
		Filename    string `json:"fileName"`
		ModName     string `json:"modName"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	mods, resp, err := CreateNewMods(w)
	if err != nil {
		return
	}

	err = mods.DownloadMod(data.DownloadURL, data.Filename, data.ModName)
	if err != nil {
		resp = fmt.Sprintf("Error downloading a mod: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = mods.ListInstalledMods()
}

func ModPortalInstallWithDepsHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		DownloadURL string `json:"downloadUrl"`
		Filename    string `json:"fileName"`
		ModName     string `json:"modName"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	mods, resp, err := CreateNewMods(w)
	if err != nil {
		return
	}

	err = mods.InstallModWithDeps(data.DownloadURL, data.Filename, data.ModName, nil)
	if err != nil {
		resp = fmt.Sprintf("Error installing mod with dependencies: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = mods.ListInstalledMods()
}

func ModPortalLoginHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	err, statusCode := factorio.FactorioLogin(data.Username, data.Password)
	w.WriteHeader(statusCode)
	if err != nil {
		resp = fmt.Sprintf("Error trying to login into Factorio: %s", err)
		log.Println(resp)
		return
	}
}

func ModPortalLoginStatusHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	var credentials factorio.Credentials
	resp, err = credentials.Load()

	if err != nil {
		resp = fmt.Sprintf("Error getting the factorio credentials: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ModPortalLogoutHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	var credentials factorio.Credentials
	err = credentials.Del()

	if err != nil {
		resp = fmt.Sprintf("Error on logging out of factorio: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = false
}

func ModPortalInstallMultipleHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data []struct {
		Name    string           `json:"name"`
		Version factorio.Version `json:"version"`
	}
	resp, err = ReadFromRequestBody(w, r, &data)
	if err != nil {
		return
	}

	modList, resp, err := CreateNewMods(w)
	if err != nil {
		return
	}

	// Mods that are built into Factorio or are DLC — cannot/should not be downloaded.
	skipMods := map[string]bool{
		"base": true, "quality": true, "elevated-rails": true, "space-age": true,
	}

	// Snapshot of currently installed mods: name → filename on disk.
	// Used to skip re-downloading mods that are already at the correct version.
	installedByName := make(map[string]string)
	for _, m := range modList.ModInfoList.Mods {
		installedByName[m.Name] = m.FileName
	}

	// Build the set of wanted mod names (excluding builtins).
	wantedNames := make(map[string]bool)
	for _, datum := range data {
		if !skipMods[datum.Name] {
			wantedNames[datum.Name] = true
		}
	}

	// Remove mods that are currently installed but not present in the save.
	for name := range installedByName {
		if !wantedNames[name] {
			if delErr := modList.DeleteMod(name); delErr != nil {
				log.Printf("ModPortalInstallMultiple: could not remove unwanted mod %s: %s", name, delErr)
			}
		}
	}

	visited := make(map[string]bool)

	for _, datum := range data {
		if skipMods[datum.Name] {
			continue
		}
		details, err, statusCode := factorio.ModPortalModDetails(datum.Name)
		if err != nil || statusCode != http.StatusOK {
			log.Printf("Warning: could not fetch portal details for %s (%d): %s – skipping", datum.Name, statusCode, err)
			continue
		}

		// Find the release matching the save's version (compare first 3 parts).
		var found = false
		for _, release := range details.Releases {
			if release.Version[0] == datum.Version[0] &&
				release.Version[1] == datum.Version[1] &&
				release.Version[2] == datum.Version[2] {
				found = true

				// Already installed at the correct version — skip the download.
				if installedByName[datum.Name] == release.FileName {
					log.Printf("ModPortalInstallMultiple: %s already at correct version (%s), skipping", datum.Name, datum.Version)
					visited[datum.Name] = true
					break
				}

				err := modList.InstallModWithDeps(release.DownloadURL, release.FileName, details.Name, visited)
				if err != nil {
					resp = fmt.Sprintf("Error downloading mod {%s}, error: %s", details.Name, err)
					log.Println(resp)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				break
			}
		}
		if !found {
			log.Printf("Warning: version %s of mod %s not found on portal – skipping", datum.Version, datum.Name)
		}
	}

	resp = modList.ListInstalledMods()
}
