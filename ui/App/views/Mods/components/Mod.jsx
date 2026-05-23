import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {
    faArrowCircleUp,
    faCheck,
    faClockRotateLeft,
    faSpinner,
    faTimes,
    faToggleOff,
    faToggleOn,
    faTrashAlt
} from "@fortawesome/free-solid-svg-icons";
import modsResource from "../../../../api/resources/mods";
import React, {useEffect, useState} from "react";
import {coerce, gt, satisfies} from "semver";
import SelectVersionForm from "./AddMod/components/SelectVersionForm";

const Mod = ({mod, factorioVersion, toggleMod, deleteMod, updateMod, addUpdatableMod, disabled = false, isSelected = false, onToggleSelect = () => {}}) => {

    const [newVersion, setNewVersion] = useState(null)
    const [icon, setIcon] = useState(faArrowCircleUp)
    const [isVersionModalOpen, setIsVersionModalOpen] = useState(false)
    const [allReleases, setAllReleases] = useState([])
    const [isLoadingVersions, setIsLoadingVersions] = useState(false)

    const openVersionModal = async () => {
        setIsLoadingVersions(true);
        try {
            const data = await modsResource.portal.info(mod.name);
            const annotated = (data.releases || []).map(release => {
                const facVer = coerce(release.info_json.factorio_version);
                return {
                    ...release,
                    compatibility: !!facVer && (
                        satisfies(factorioVersion, "~" + facVer.version) ||
                        (satisfies(factorioVersion, "~1.0.0") && satisfies(facVer, "~0.18.0"))
                    )
                };
            });
            setAllReleases(annotated);
            setIsVersionModalOpen(true);
        } finally {
            setIsLoadingVersions(false);
        }
    };

    const installVersion = release => {
        const ver = coerce(release.version);
        return updateMod({
            downloadUrl: release.download_url,
            fileName: release.file_name,
            modName: mod.name,
            version: ver ? ver.version : release.version
        });
    };

    useEffect(() => {
        if (!disabled && !mod.built_in) {
            (async () => {
                const data = await modsResource.portal.info(mod.name)

                //get newest COMPATIBLE release
                let newestRelease;
                data.releases.forEach(release => {
                    const releaseVer = coerce(release.version);
                    const modVer = coerce(mod.version);
                    const facVer = coerce(release.info_json.factorio_version);
                    if (!releaseVer || !modVer || !facVer) return;
                    if (
                        gt(releaseVer, modVer) && (
                            satisfies(factorioVersion, "~" + facVer.version) ||
                            (
                                satisfies(factorioVersion, "~1.0.0") &&
                                satisfies(facVer, "~0.18.0")
                            )
                        )
                    ) {
                        if (!newestRelease) {
                            newestRelease = release;
                        } else {
                            const nVer = coerce(newestRelease.version);
                            if (nVer && gt(releaseVer, nVer)) newestRelease = release;
                        }
                    }
                });

                if (newestRelease && newestRelease.version !== mod.version) {
                    const installableVersion = {
                        downloadUrl: newestRelease.download_url,
                        fileName: newestRelease.file_name,
                        modName: mod.name,
                        version: coerce(newestRelease.version).version
                    }
                    setNewVersion(installableVersion);
                    if (addUpdatableMod !== null) {
                        addUpdatableMod(installableVersion)
                    }
                } else {
                    setNewVersion(null);
                }

            })();
        }
    }, [mod]);

    return (
        <>
        <SelectVersionForm
            isOpen={isVersionModalOpen}
            releases={allReleases}
            install={installVersion}
            close={() => setIsVersionModalOpen(false)}
        />
        <tr className="py-1">
            <td className="pr-4">
                {mod.built_in
                    ? <span>{mod.name} <span className="text-xs italic text-gray-500">Built-in</span></span>
                    : mod.title
                }
            </td>
            <td className="pr-4">
                {
                    disabled
                        ?

                        mod.enabled
                            ? <FontAwesomeIcon className="text-green" icon={faCheck}/>
                            : <FontAwesomeIcon className="text-red" icon={faTimes}/>
                        :
                        mod.enabled
                            ? <FontAwesomeIcon className="cursor-pointer hover:text-green-light text-green"
                                               icon={faToggleOn}
                                               onClick={() => toggleMod(mod.name)}/>
                            :
                            <FontAwesomeIcon className="cursor-pointer hover:text-red-light text-red"
                                             icon={faToggleOff}
                                             onClick={() => toggleMod(mod.name)}/>
                }
            </td>
            <td className="pr-4">
                {mod.built_in ? <span className="text-xs italic text-gray-500">—</span> : mod.compatibility
                    ? <FontAwesomeIcon className="text-green" icon={faCheck}/>
                    : <FontAwesomeIcon className="text-red" icon={faTimes}/>
                }
            </td>
            <td className="pr-4">
                {mod.built_in
                    ? <span className="text-xs italic text-gray-500">DLC</span>
                    : <>
                        {mod.version}
                        {!disabled && newVersion && <>
                            <input
                                type="checkbox"
                                className="ml-2 cursor-pointer"
                                checked={isSelected}
                                onChange={onToggleSelect}
                                title="Select for bulk update"
                            />
                            <span className="text-orange ml-1">→ {newVersion.version}</span>
                            <FontAwesomeIcon spin={icon === faSpinner}
                                                        onClick={() => {
                                                            setIcon(faSpinner)
                                                            updateMod(newVersion)
                                                                .finally(() => setIcon(faArrowCircleUp))
                                                        }}
                                                        className="hover:text-orange cursor-pointer ml-1"
                                                        icon={icon}/>
                        </>}
                        {!disabled &&
                            <FontAwesomeIcon
                                icon={isLoadingVersions ? faSpinner : faClockRotateLeft}
                                spin={isLoadingVersions}
                                onClick={openVersionModal}
                                title="Select version"
                                className="cursor-pointer ml-2 text-gray-400 hover:text-white"
                            />
                        }
                    </>
                }
            </td>
            <td className="pr-4">{mod.built_in ? <span className="text-xs italic text-gray-500">—</span> : mod.factorio_version}</td>
            {
                !disabled && !mod.built_in &&
                <td className="pr-4">
                    <FontAwesomeIcon className={"text-red cursor-pointer hover:text-red-light"}
                                     onClick={() => deleteMod(mod.name)} icon={faTrashAlt}/>
                </td>
            }
            {
                !disabled && mod.built_in && <td/>
            }
        </tr>
        </>
    )
}

export default Mod;

