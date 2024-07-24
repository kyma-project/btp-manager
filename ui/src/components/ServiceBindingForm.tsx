import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
    ServiceInstanceBinding,
} from "../shared/models";
import { useEffect, useState } from "react";
import axios from "axios";
import api from "../shared/api";

import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';

function ServiceBindingForm(props: any) {
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const [serviceInstanceID, setServiceInstanceID] = useState('');
    const [name, setName] = useState('');

    const handleCreate = () => {
        setLoading(true)
        axios
            .post<ServiceInstanceBinding>(api("service-bindings"), 
            {name: name, serviceInstanceID: serviceInstanceID})
            .then((response) => {
                setLoading(false);
                // setServiceInstances(response.data);
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
            });
    }

    useEffect(() => {
        if (!Ok(props.serviceInstanceID)) {
            return;
        }

        setName(generateUUID())
        setServiceInstanceID(props.serviceInstanceID)

    }, [props.serviceInstanceID]);

    const renderData = () => {

        return (
            <>
                <ui5.Form>
                    <ui5.FormItem label={<ui5.Label required>Name</ui5.Label>}>
                        <ui5.Input
                            style={{ width: "100vw" }}
                            required
                            value={name}
                        />
                    </ui5.FormItem>
                    <ui5.FormItem label="Service ID">
                        <ui5.Input
                            style={{ width: "100vw"}}
                            required
                            title="Service Instance ID"
                            value={serviceInstanceID}
                        />
                    </ui5.FormItem>
                    <ui5.FormItem>
                        <ui5.Button onClick={handleCreate}>Create</ui5.Button>
                    </ui5.FormItem>
                </ui5.Form>
            </>
        )
    }
    // @ts-ignore
    return <>{renderData()}</>;
}

function generateUUID() : string {
    return window.crypto.randomUUID().substring(0, 4)
}

export default ServiceBindingForm;