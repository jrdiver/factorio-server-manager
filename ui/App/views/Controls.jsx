import React, {useEffect, useState} from "react";
import Panel from "../components/Panel";
import Button from "../components/Button";
import server from "../../api/resources/server";
import savesResource from "../../api/resources/saves";
import modsResource from "../../api/resources/mods";
import {useForm} from "react-hook-form";
import Select from "../components/Select";
import Input from "../components/Input";
import Error from "../components/Error";

const Controls = ({serverStatus, refreshServerStatus}) => {

    const factorioVersion = serverStatus?.fac_version ? serverStatus.fac_version : 'Unknown';
    const [saves, setSaves] = useState([]);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isStopping, setIsStopping] = useState(false);
    const [isStarting, setIsStarting] = useState(false);
    const [isKilling, setIsKilling] = useState(false);
    // modWarnings: [{type: 'missing'|'disabled'|'version', name, saveVer, instVer, checked}]
    const [modWarnings, setModWarnings] = useState([]);
    const [isSyncing, setIsSyncing] = useState(false);

    const { handleSubmit, reset, register, watch, formState: {errors} } = useForm();
    const selectedSave = watch('save');

    const startServer = async (data) => {
        setIsStarting(true);
        try {
            await server.start(data.ip, parseInt(data.port), data.save);
        } finally {
            setIsStarting(false);
            await refreshServerStatus();
        }
    }

    const stopServer = async () => {
        setIsStopping(true);
        try {
            await server.stop();
        } finally {
            setIsStopping(false);
            await refreshServerStatus();
        }
    }

    const killServer = async () => {
        setIsKilling(true);
        try {
            await server.kill();
        } finally {
            setIsKilling(false);
            await refreshServerStatus();
        }
    }

    // Always fetch current status on mount so the page is fresh
    useEffect(() => {
        refreshServerStatus();
    }, []);

    const checkModMismatches = (saveName) => {
        if (!saveName) { setModWarnings([]); return; }
        Promise.all([
            savesResource.mods(saveName),
            modsResource.installed()
        ]).then(([saveHeader, installedData]) => {
            const saveMods = saveHeader.mods || [];
            // installed() returns the array directly (not wrapped in {mods:[...]})
            const installedMods = Array.isArray(installedData) ? installedData : [];
            const skipMods = new Set(['base', 'quality', 'elevated-rails', 'space-age']);
            const warnings = [];
            for (const saveMod of saveMods) {
                if (skipMods.has(saveMod.name)) continue;
                const installed = installedMods.find(m => m.name === saveMod.name);
                const saveVer = saveMod.version.split('.').slice(0, 3).join('.');
                if (!installed) {
                    warnings.push({type: 'missing', name: saveMod.name, saveVer, instVer: null, checked: true});
                } else if (!installed.enabled) {
                    warnings.push({type: 'disabled', name: saveMod.name, saveVer, instVer: null, checked: true});
                } else {
                    const instVer = installed.version.split('.').slice(0, 3).join('.');
                    if (saveVer !== instVer) {
                        warnings.push({type: 'version', name: saveMod.name, saveVer, instVer, checked: true});
                    }
                }
            }
            setModWarnings(warnings);
        }).catch(() => {});
    };

    const toggleWarning = (name) =>
        setModWarnings(prev => prev.map(w => w.name === name ? {...w, checked: !w.checked} : w));

    const toggleAllWarnings = (checked) =>
        setModWarnings(prev => prev.map(w => ({...w, checked})));

    const syncSelected = async () => {
        const selected = modWarnings.filter(w => w.checked);
        if (selected.length === 0) return;
        setIsSyncing(true);
        for (const w of selected) {
            try {
                if (w.type === 'disabled') {
                    await modsResource.toggle(w.name);
                } else {
                    // missing or version mismatch — need portal
                    const info = await modsResource.portal.info(w.name);
                    const releases = info.releases || [];
                    const release = releases.find(r => r.version.split('.').slice(0, 3).join('.') === w.saveVer);
                    if (!release) {
                        window.flash(`No portal release found for ${w.name} v${w.saveVer}`, 'red');
                        continue;
                    }
                    if (w.type === 'version') {
                        await modsResource.delete(w.name);
                    }
                    await modsResource.portal.installWithDeps(release.download_url, release.file_name, w.name);
                }
            } catch (e) {
                window.flash(`Failed to sync ${w.name}: ${e?.message || 'unknown error'}`, 'red');
            }
        }
        setIsSyncing(false);
        checkModMismatches(selectedSave);
    };

    // Check for mod mismatches when selected save changes
    useEffect(() => {
        if (!selectedSave || serverStatus?.running) { setModWarnings([]); return; }
        checkModMismatches(selectedSave);
    }, [selectedSave]);

    useEffect(() => {
        savesResource.list(true)
            .then(res => {
                setSaves(res);
                if (res.length > 0) {
                    setIsDisabled(undefined);
                }
                reset();
            });
    }, [])

    return (
        <form onSubmit={handleSubmit(startServer)}>
        <Panel
            title="Server Status"
            content={
                <div className="lg:flex">
                    { serverStatus?.running
                        ? <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{serverStatus.running ? 'Running' : 'Stopped'}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">IP</div>
                                <div>{serverStatus.bindip}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Port</div>
                                <div>{serverStatus.port}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Factorio Version</div>
                                <div>{factorioVersion}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Save</div>
                                <div>{serverStatus.savefile}</div>
                            </div>
                        </>
                        : <>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Status</div>
                                <div>{serverStatus.running ? 'Running' : 'Stopped'}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">IP</div>
                                <Input
                                    defaultValue={"0.0.0.0"}
                                    disabled={isDisabled}
                                    register={register('ip',{required: true, pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/})}
                                />
                                <Error error={errors.ip} message="IP is required and must be valid."/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">Port</div>
                                <Input
                                    type="number"
                                    min={1}
                                    max={65535}
                                    defaultValue={"34197"}
                                    disabled={isDisabled}
                                    register={register('port',{required: true, min: 1, max: 65535})}
                                />
                                <Error error={errors.port} message="Port is required within range 1-65535"/>
                            </div>
                            <div className="lg:w-1/5 mb-2 mr-0 lg:mr-4">
                                <div className="font-bold">Factorio Version</div>
                                <div>{factorioVersion}</div>
                            </div>
                            <div className="lg:w-1/5 mb-2">
                                <div className="font-bold">Save</div>
                                <div className="relative">
                                    <Select
                                        register={register('save',{required: true})}
                                        defaultValue={saves.find((save) => save.name.startsWith('Load Latest'))?.name}
                                        disabled={isDisabled}
                                        options={saves.map(save => new Object({
                                            value: save.name,
                                            name: save.name
                                        }))}
                                    />
                                    <Error error={errors.save} message="Save is required and must be valid."/>
                                </div>
                            </div>
                        </>
                    }
                </div>
            }
            actions={
                <div className="md:flex">
                    {serverStatus?.running
                        ? <>
                            <Button onClick={stopServer} isLoading={isStopping} isDisabled={isKilling} size="sm" className="w-full md:w-auto mb-2 md:mb-0 md:mr-2" type="default">Save & Stop Server</Button>
                            <Button onClick={killServer} isLoading={isKilling} isDisabled={isStopping} size="sm" type="danger" className="w-full md:w-auto">Kill Server</Button>
                        </>
                        : <Button isSubmit={true} isDisabled={isDisabled} isLoading={isStarting} size="sm" type="success" className="w-full md:w-auto">Start Server</Button>
                    }
                </div>
            }
        />
        {modWarnings.length > 0 &&
            <div className="mt-2 p-3 border rounded text-sm" style={{backgroundColor:'#fffbeb', borderColor:'#f59e0b', color:'#92400e'}}>
                <div className="font-bold mb-2">⚠ Mod mismatch — select items to sync:</div>
                <table className="w-full mb-2">
                    <thead>
                        <tr className="text-left border-b" style={{borderColor:'#f59e0b'}}>
                            <th className="pb-1 pr-2">
                                <input type="checkbox"
                                    checked={modWarnings.every(w => w.checked)}
                                    onChange={e => toggleAllWarnings(e.target.checked)}
                                />
                            </th>
                            <th className="pb-1 pr-4">Mod</th>
                            <th className="pb-1 pr-4">Issue</th>
                            <th className="pb-1">Details</th>
                        </tr>
                    </thead>
                    <tbody>
                        {modWarnings.map(w => (
                            <tr key={w.name} className="border-b border-dashed" style={{borderColor:'#fcd34d'}}>
                                <td className="py-1 pr-2">
                                    <input type="checkbox" checked={w.checked} onChange={() => toggleWarning(w.name)} />
                                </td>
                                <td className="py-1 pr-4 font-mono">{w.name}</td>
                                <td className="py-1 pr-4">
                                    {w.type === 'missing'  && <span className="font-semibold text-red-700">Not installed</span>}
                                    {w.type === 'disabled' && <span className="font-semibold text-yellow-700">Disabled</span>}
                                    {w.type === 'version'  && <span className="font-semibold text-orange-700">Wrong version</span>}
                                </td>
                                <td className="py-1 text-xs">
                                    {w.type === 'missing'  && `Need v${w.saveVer}`}
                                    {w.type === 'disabled' && `Enable v${w.saveVer}`}
                                    {w.type === 'version'  && `installed: v${w.instVer} → need v${w.saveVer}`}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
                {modWarnings.some(w => w.checked && w.type !== 'disabled') &&
                    <p className="text-xs mb-2 italic">⚠ Downloading mods requires portal credentials (set in Mods → Add Mod).</p>
                }
                <Button
                    onClick={syncSelected}
                    isLoading={isSyncing}
                    isDisabled={!modWarnings.some(w => w.checked) || isSyncing}
                    size="sm"
                    type="warning"
                >
                    Sync Selected
                </Button>
            </div>
        }
        </form>
    )
};

export default Controls;
