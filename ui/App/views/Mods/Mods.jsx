import Panel from "../../components/Panel";
import React, {useEffect, useState} from "react";
import modsResource from "../../../api/resources/mods";
import Button from "../../components/Button";
import server from "../../../api/resources/server";
import TabControl from "../../components/Tabs/TabControl";
import Tab from "../../components/Tabs/Tab";
import AddMod from "./components/AddMod/AddMod";
import UploadMod from "./components/UploadMod";
import LoadMods from "./components/LoadMods";
import Fuse from "fuse.js";
import CreateModPack from "./components/CreateModPack";
import ModPack from "./components/ModPack";
import ModList from "./components/ModList";

const Mods = ({serverStatus}) => {

    const [installedMods, setInstalledMods] = useState([]);
    const [modPacks, setModPacks] = useState([])
    const [factorioVersion, setFactorioVersion] = useState(null);
    const [fuse, setFuse] = useState(undefined);
    const [isDeletingAllMods, setIsDeletingAllMods] = useState(false);
    const [isUpdatingAllMods, setIsUpdatingAllMods] = useState(false);
    const [isUpdatingSelectedMods, setIsUpdatingSelectedMods] = useState(false);
    const [updatableMods, setUpdatableMods] = useState([]);
    const [selectedModNames, setSelectedModNames] = useState({});
    const [updateProgress, setUpdateProgress] = useState({current: 0, total: 0});

    const addUpdatableMod = mod => {
        // Upsert by modName — prevents duplicates when useEffect re-fires after each refresh.
        setUpdatableMods(prev => [...prev.filter(m => m.modName !== mod.modName), mod]);
        setSelectedModNames(names => ({...names, [mod.modName]: true}));
    };

    const toggleModSelection = name => {
        setSelectedModNames(prev => {
            const next = {...prev};
            next[name] ? delete next[name] : (next[name] = true);
            return next;
        });
    };

    const fetchInstalledMods = () => {
        modsResource.installed()
            .then(mods => setInstalledMods(
                [...mods].sort((a, b) => a.name.localeCompare(b.name, undefined, {sensitivity: 'base'}))
            ));
    };

    const fetchModPacks = () => {
        modsResource.packs.list()
            .then(setModPacks)
    }

    const deleteAllMods = () => {
        setIsDeletingAllMods(true);
        modsResource.deleteAll()
            .then(fetchInstalledMods)
            .finally(() => setIsDeletingAllMods(false))
    }

    const updateAllMods = async () => {
        setIsUpdatingAllMods(true);
        setUpdateProgress({current: 0, total: updatableMods.length});
        for (let i = 0; i < updatableMods.length; i++) {
            await modsResource.update(updatableMods[i]).catch(err => console.error('Update failed:', err));
            setUpdateProgress({current: i + 1, total: updatableMods.length});
        }
        setUpdatableMods([]);
        setSelectedModNames({});
        fetchInstalledMods();
        setIsUpdatingAllMods(false);
        setUpdateProgress({current: 0, total: 0});
    }

    const updateSelectedMods = async () => {
        setIsUpdatingSelectedMods(true);
        const toUpdate = updatableMods.filter(m => selectedModNames[m.modName]);
        setUpdateProgress({current: 0, total: toUpdate.length});
        for (let i = 0; i < toUpdate.length; i++) {
            await modsResource.update(toUpdate[i]).catch(err => console.error('Update failed:', err));
            setUpdateProgress({current: i + 1, total: toUpdate.length});
        }
        setUpdatableMods([]);
        setSelectedModNames({});
        fetchInstalledMods();
        setIsUpdatingSelectedMods(false);
        setUpdateProgress({current: 0, total: 0});
    }

    useEffect(() => {
        server.factorioVersion()
            .then(data => {
                setFactorioVersion(data.base_mod_version)
                fetchInstalledMods();
                fetchModPacks();
            })

        // fetch list of mods
        modsResource.portal.list()
            .then(res => {
                setFuse(new Fuse(res.results, {
                    keys: [
                        {
                            "name": "name",
                            weight: 2
                        },
                        {
                            "name": "title",
                            weight: 1
                        }
                    ],
                    minMatchCharLength: 3
                }));
            });

    }, []);

    const toggleMod = modName => {
        return modsResource
            .toggle(modName)
            .then(fetchInstalledMods)
    }

    const deleteMod = modName => {
        return modsResource
            .delete(modName)
            .then(fetchInstalledMods)
    }

    const updateMod = version => {
        return modsResource
            .update(version)
            .then(fetchInstalledMods)
    }

    let disabled = serverStatus.running

    return (
        <div>
            {disabled ?
                <Panel className="mb-6"
                       content={
                           <div className="text-red font-bold text-xl">
                               Changing mods is disabled while the server is running!
                           </div>
                       }
                />
                :
                <TabControl>
                    <Tab title="Install Mod">
                        <AddMod refetchInstalledMods={fetchInstalledMods} fuse={fuse}/>
                    </Tab>
                    <Tab title="Upload Mod">
                        <UploadMod refetchInstalledMods={fetchInstalledMods}/>
                    </Tab>
                    <Tab title="Load Mods from Save">
                        <LoadMods refreshMods={fetchInstalledMods}/>
                    </Tab>
                </TabControl>
            }
            <Panel
                title="Mods"
                className="mb-6"
                content={
                    <ModList addUpdatableMod={addUpdatableMod}
                             toggleMod={toggleMod}
                             updateMod={updateMod}
                             deleteMod={deleteMod}
                             mods={installedMods}
                             factorioVersion={factorioVersion}
                             disabled={disabled}
                             selectedModNames={selectedModNames}
                             toggleModSelection={toggleModSelection}
                    />
                }
                actions={
                    <>
                        {
                            !disabled &&
                            <Button size="sm" className="mr-2" type="danger" isLoading={isDeletingAllMods}
                                    onClick={deleteAllMods}>Delete all Mods</Button> &&
                            <Button size="sm" className="mr-2" isLoading={isUpdatingAllMods}
                                    onClick={updateAllMods}>Update all Mods</Button>
                        }                        {
                            !disabled && Object.keys(selectedModNames).length > 0 &&
                            <Button size="sm" className="mr-2" isLoading={isUpdatingSelectedMods}
                                    onClick={updateSelectedMods}>
                                Update Selected ({Object.keys(selectedModNames).length})
                            </Button>
                        }                        {
                            (isUpdatingAllMods || isUpdatingSelectedMods) && updateProgress.total > 0 &&
                            <span className="text-sm mr-2 text-white">
                                {updateProgress.current}/{updateProgress.total} updated
                            </span>
                        }                        <a className="bg-gray-light py-1 px-2 hover:glow-orange hover:bg-orange inline-block accentuated text-black font-bold"
                           href={modsResource.downloadAllURL}>Download all Mods</a>
                    </>
                }
            />

            <Panel
                title="Mod packs"
                className="mb-6"
                content={
                    modPacks.map(
                        (pack, i) =>
                            <ModPack factorioVersion={factorioVersion}
                                     key={i}
                                     modPack={pack}
                                     reloadMods={fetchInstalledMods}
                                     reloadModPacks={fetchModPacks}
                                     disabled={disabled}
                            />
                    )
                }
                actions={
                    <CreateModPack onSuccess={fetchModPacks}/>
                }
            />
        </div>
    )
}

export default Mods;
