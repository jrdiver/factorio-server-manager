import React, {useEffect, useState} from "react";
import savesResource from "../../../../api/resources/saves";
import Select from "../../../components/Select";
import Label from "../../../components/Label";
import {useForm} from "react-hook-form";
import Button from "../../../components/Button";
import modsResource from "../../../../api/resources/mods";
import modResource from "../../../../api/resources/mods";
import FactorioLogin from "./AddMod/components/FactorioLogin";
import ConfirmDialog from "../../../components/ConfirmDialog";

const LoadMods = ({refreshMods}) => {

    const [saves, setSaves] = useState([]);
    const {register, reset, handleSubmit} = useForm();
    const [isLoading, setIsLoading] = useState(false);
    const [isDisabled, setIsDisabled] = useState(true);
    const [isFactorioAuthenticated, setIsFactorioAuthenticated] = useState(false);
    const [loadModsData, setLoadModsData] = useState(undefined);
    const [syncStatus, setSyncStatus] = useState('');
    const [syncProgress, setSyncProgress] = useState(null); // {current, total} or null

    useEffect(() => {
        (async () => {
            setIsFactorioAuthenticated(await modResource.portal.status())

            const s = await savesResource.list()
            setSaves(s);
            if (s.length > 0) {
                setIsDisabled(false);
            }
            reset();
        })();
    }, []);

    const loadModsRequested = data => {
        setIsLoading(true);
        setLoadModsData(data);
    }

    const loadMods = async data => {
        // Fetch the save's mod list to check what's needed.
        setSyncStatus('Reading save file…');
        setSyncProgress(null);
        const saveHeader = await savesResource.mods(data.save).catch(() => {
            setIsLoading(false);
            setLoadModsData(undefined);
            setSyncStatus('');
        });
        if (!saveHeader) return;

        if (!saveHeader.mods || saveHeader.mods.length === 0) {
            window.flash(`Save file "${data.save}" returned no mods — refusing to wipe current mods.`, "red");
            setIsLoading(false);
            setLoadModsData(undefined);
            setSyncStatus('');
            return;
        }

        // Phase 1: let the backend delete unwanted mods and return what needs downloading.
        setSyncStatus('Calculating changes…');
        let toInstall;
        try {
            toInstall = await modResource.portal.syncPrepare(saveHeader.mods);
        } catch {
            setIsLoading(false);
            setLoadModsData(undefined);
            setSyncStatus('');
            return;
        }

        if (toInstall.length === 0) {
            window.flash(`Mods already match save file ${data.save} — nothing to download.`, "green");
            refreshMods();
            setIsLoading(false);
            setLoadModsData(undefined);
            setSyncStatus('');
            return;
        }

        // Phase 2: install each mod one-by-one so we can show progress.
        const total = toInstall.length;
        setSyncProgress({current: 0, total});
        setSyncStatus('');

        let failed = false;
        for (let i = 0; i < toInstall.length; i++) {
            const mod = toInstall[i];
            setSyncProgress({current: i, total});
            try {
                await modResource.portal.installWithDeps(mod.downloadUrl, mod.fileName, mod.name);
            } catch {
                failed = true;
                break;
            }
        }

        if (!failed) {
            setSyncProgress({current: total, total});
            window.flash(`Mods synced from save file ${data.save}.`, "green");
        }

        refreshMods();
        setIsLoading(false);
        setLoadModsData(undefined);
        setSyncStatus('');
        setSyncProgress(null);
    }

    return isFactorioAuthenticated
        ? <form onSubmit={handleSubmit(loadModsRequested)}>
            <Label text="Save" htmlFor="save"/>
            <Select
                register={register('save')}
                className="mb-4"
                disabled={isDisabled}
                options={saves?.map(save => new Object({
                    name: save.name,
                    value: save.name
                }))}
            />
            <Button isSubmit={true} isDisabled={isDisabled} isLoading={isLoading}>Load</Button>
            {syncStatus && <p className="mt-2 text-sm text-gray-400">{syncStatus}</p>}
            {syncProgress && (
                <p className="mt-2 text-sm text-white">{syncProgress.current}/{syncProgress.total} installed</p>
            )}
            <ConfirmDialog
                title="Load Mods from Save"
                content={`Syncing mods to match save "${loadModsData?.save}": mods not in the save will be removed, and any missing or outdated mods will be downloaded. Already-installed mods at the correct version will be skipped.`}
                isOpen={loadModsData !== undefined}
                close={() => {
                    setIsLoading(false);
                    setLoadModsData(undefined);
                }}
                onSuccess={() => loadMods(loadModsData)}
            />
        </form>
        : <FactorioLogin setIsFactorioAuthenticated={setIsFactorioAuthenticated}/>
}

export default LoadMods;
