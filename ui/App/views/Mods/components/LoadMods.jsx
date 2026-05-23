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

        const total = saveHeader.mods.length;
        setSyncStatus(`Syncing ${total} mods from save…`);

        await modResource.portal.installMultiple(saveHeader.mods)
            .then(() => {
                window.flash(`Mods synced from save file ${data.save}.`, "green");
            })
            .catch(() => {
                // The Axios interceptor already flashed the real error from the server.
            })
            .finally(() => {
                refreshMods();
                setIsLoading(false);
                setLoadModsData(undefined);
                setSyncStatus('');
            });
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
