package factorio

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/lockfile"
)

type Mods struct {
	ModSimpleList ModSimpleList `json:"mod_simple_list"`
	ModInfoList   ModInfoList   `json:"mod_info_list"`
}
type ModsResult struct {
	ModInfo
	Enabled bool `json:"enabled"`
	BuiltIn bool `json:"built_in"`
}
type ModsResultList struct {
	ModsResult []ModsResult `json:"mods"`
}

var FileLock lockfile.FileLock = lockfile.NewLock()

func NewMods(destination string) (Mods, error) {
	var err error
	var mods Mods

	mods.ModSimpleList, err = newModSimpleList(destination)
	if err != nil {
		log.Printf("error on creating newModSimpleList: %s", err)
		return mods, err
	}

	mods.ModInfoList, err = newModInfoList(destination)
	if err != nil {
		log.Printf("error on creating newModInfoList: %s", err)
		return mods, err
	}

	return mods, nil
}

func (mods *Mods) ListInstalledMods() ModsResultList {
	result := ModsResultList{make([]ModsResult, 0)}

	for _, modInfo := range mods.ModInfoList.Mods {
		var modsResult ModsResult
		modsResult.Name = modInfo.Name
		modsResult.FileName = modInfo.FileName
		modsResult.Author = modInfo.Author
		modsResult.Title = modInfo.Title
		modsResult.Version = modInfo.Version
		modsResult.FactorioVersion = modInfo.FactorioVersion
		modsResult.Compatibility = modInfo.Compatibility

		for _, simpleMod := range mods.ModSimpleList.Mods {
			if simpleMod.Name == modsResult.Name {
				modsResult.Enabled = simpleMod.Enabled
				break
			}
		}

		result.ModsResult = append(result.ModsResult, modsResult)
	}

	// Include mods from mod-list.json that have no zip file (built-in/DLC).
	// Skip "base" — it is always required and cannot meaningfully be disabled.
	infoNames := make(map[string]bool)
	for _, m := range mods.ModInfoList.Mods {
		infoNames[m.Name] = true
	}
	for _, simpleMod := range mods.ModSimpleList.Mods {
		if simpleMod.Name == "base" || infoNames[simpleMod.Name] {
			continue
		}
		// Only the four known DLC entries should appear as "built-in".
		// Any other mod missing a zip file is an orphaned entry; hide it.
		if !builtinMods[simpleMod.Name] {
			continue
		}
		result.ModsResult = append(result.ModsResult, ModsResult{
			ModInfo: ModInfo{Name: simpleMod.Name},
			Enabled: simpleMod.Enabled,
			BuiltIn: true,
		})
	}

	return result
}

func (mods *Mods) DeleteMod(modName string) error {
	var err error

	err = mods.ModInfoList.deleteMod(modName)
	if err != nil {
		log.Printf("error when deleting mod in ModInfoList: %s", err)
		return err
	}

	err = mods.ModSimpleList.deleteMod(modName)
	if err != nil {
		log.Printf("error when deleting mod in ModSimpleList: %s", err)
		return err
	}

	return nil
}

func (mods *Mods) createMod(modName string, fileName string, fileRc io.Reader) error {
	var err error

	//check if mod already exists and delete it
	if mods.ModSimpleList.CheckModExists(modName) {
		err = mods.ModInfoList.deleteMod(modName)
		if err != nil {
			// The mod is in mod-list.json but has no zip file (orphaned entry from a
			// failed/partial install). Nothing to remove from disk — proceed to create.
			log.Printf("createMod: no zip found for existing %s entry — proceeding to install: %s", modName, err)
		}
	}

	//create new mod
	err = mods.ModInfoList.createMod(modName, fileName, fileRc)
	if err != nil {
		log.Printf("error on creating mod-file: %s", err)

		// removing mod completely
		err2 := mods.ModSimpleList.deleteMod(modName)
		if err2 != nil {
			log.Printf("error deleting mod from modSimpleList: %s", err2)
		}

		return err
	}

	// also add to ModSimpleList if not there yet
	if !mods.ModSimpleList.CheckModExists(modName) {
		err = mods.ModSimpleList.createMod(modName)
		if err != nil {
			log.Printf("error creating mod in modSimpleList: %s", err)
			return err
		}
	}

	return nil
}

func (mods *Mods) DownloadMod(url string, filename string, modId string) error {
	var err error

	var credentials Credentials
	status, err := credentials.Load()
	if err != nil {
		log.Printf("error loading credentials: %s", err)
		return err
	}
	if status == false {
		log.Printf("error: credentials are invalid")
		return errors.New("error: credentials are invalid")
	}

	//download the mod from the mod portal api
	completeUrl := "https://mods.factorio.com" + url + "?username=" + credentials.Username + "&token=" + credentials.Userkey

	response, err := http.Get(completeUrl)
	if err != nil {
		log.Printf("error on downloading mod: %s", err)
		return err
	}

	log.Printf("download complete\n StatusCode: %d\n Status: %s", response.StatusCode, response.Status)

	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Printf("StatusCode: %d", response.StatusCode)

		return errors.New("Statuscode not 200: " + fmt.Sprint(response.StatusCode))
	}

	err = mods.createMod(modId, filename, response.Body)
	if err != nil {
		log.Printf("error when creating Mod: %s", err)
		return err
	}

	log.Printf("completed copying the response.Body")

	//done everything is made inside the createMod

	return nil
}

func (mods *Mods) UploadMod(file multipart.File, header *multipart.FileHeader) error {
	var err error

	if filepath.Ext(header.Filename) != ".zip" {
		log.Print("The uploaded file wasn't a zip-file")
		return errors.New("the uploaded file wasn't a zip-file")
	}

	fileByteArray, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("error reading file: %s", err)
		return err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(fileByteArray), int64(len(fileByteArray)))
	if err != nil {
		log.Printf("Uploaded file could not put into zip.Reader: %s", err)
		return err
	}

	var modInfo ModInfo
	err = modInfo.getModInfo(zipReader)
	if err != nil {
		log.Printf("Error in getModInfo: %s", err)
		return err
	}

	err = mods.createMod(modInfo.Name, header.Filename, bytes.NewReader(fileByteArray))
	if err != nil {
		log.Printf("error on creating Mod: %s", err)
		return err
	}

	return nil
}

func (mods *Mods) UpdateMod(modName string, url string, filename string) error {
	// Find the current file so we can back it up before overwriting.
	var oldFilePath, backupPath string
	for _, m := range mods.ModInfoList.Mods {
		if m.Name == modName {
			oldFilePath = filepath.Join(mods.ModInfoList.Destination, m.FileName)
			break
		}
	}
	if oldFilePath != "" {
		backupPath = oldFilePath + ".bak"
		if err := copyFileAtomic(oldFilePath, backupPath); err != nil {
			log.Printf("updateMod: could not back up %s before update: %s — proceeding without rollback", modName, err)
			backupPath = ""
		}
	}

	err := mods.DownloadMod(url, filename, modName)
	if err != nil {
		log.Printf("updateMod: error downloading new version of %s: %s", modName, err)
		if backupPath != "" {
			if restoreErr := os.Rename(backupPath, oldFilePath); restoreErr != nil {
				log.Printf("updateMod: WARN — could not restore backup for %s: %s", modName, restoreErr)
			} else {
				log.Printf("updateMod: restored previous version of %s after failed update", modName)
				_ = mods.ModInfoList.listInstalledMods()
			}
		}
		return err
	}

	if backupPath != "" {
		_ = os.Remove(backupPath)
	}
	return nil
}

// copyFileAtomic copies src to dst, creating dst if it does not exist.
func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// builtinMods are part of the Factorio game and are not available as portal downloads.
var builtinMods = map[string]bool{
	"base":           true,
	"quality":        true,
	"elevated-rails": true,
	"space-age":      true,
}

// InstallModWithDeps downloads a mod and recursively installs any required dependencies
// that are not already present. Pass nil for visited on the initial call.
func (mods *Mods) InstallModWithDeps(downloadURL, filename, modName string, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}
	if visited[modName] {
		return nil
	}
	visited[modName] = true

	// Fetch full mod info to get the dependency list for this specific release.
	modDetails, err, _ := ModPortalModDetailsFull(modName)
	if err != nil {
		log.Printf("InstallModWithDeps: could not fetch full info for %s: %s – installing without deps", modName, err)
		return mods.DownloadMod(downloadURL, filename, modName)
	}

	// Find the release that matches the requested file.
	var targetRelease *ModPortalFullRelease
	for i, r := range modDetails.Releases {
		if r.FileName == filename || r.DownloadURL == downloadURL {
			targetRelease = &modDetails.Releases[i]
			break
		}
	}

	if targetRelease != nil {
		for _, dep := range targetRelease.InfoJSON.Dependencies {
			depName, required := parseDependencyName(dep)
			if !required || depName == "" || builtinMods[depName] {
				continue
			}
			if mods.ModSimpleList.CheckModExists(depName) {
				continue
			}
			// Fetch dep info and install its latest compatible release.
			depDetails, depErr, _ := ModPortalModDetailsFull(depName)
			if depErr != nil {
				log.Printf("InstallModWithDeps: dependency %s not found on portal: %s", depName, depErr)
				continue
			}
			var latestRelease *ModPortalFullRelease
			for i := len(depDetails.Releases) - 1; i >= 0; i-- {
				if depDetails.Releases[i].Compatibility {
					latestRelease = &depDetails.Releases[i]
					break
				}
			}
			if latestRelease == nil && len(depDetails.Releases) > 0 {
				latestRelease = &depDetails.Releases[len(depDetails.Releases)-1]
			}
			if latestRelease != nil {
				log.Printf("InstallModWithDeps: installing required dependency %s %s", depName, latestRelease.Version)
				if depErr = mods.InstallModWithDeps(latestRelease.DownloadURL, latestRelease.FileName, depName, visited); depErr != nil {
					log.Printf("InstallModWithDeps: error installing dependency %s: %s", depName, depErr)
				}
			}
		}
	}

	return mods.DownloadMod(downloadURL, filename, modName)
}
