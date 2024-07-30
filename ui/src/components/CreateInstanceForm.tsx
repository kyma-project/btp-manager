import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
    ServiceInstance,
} from "../shared/models";
import { useEffect, useState } from "react";
import axios from "axios";
import api from "../shared/api";

import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';

function CreateInstanceForm(props: any) {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [loading, setLoading] = useState(true);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [error, setError] = useState(null);

    const [name, setName] = useState('');
    const [params] = useState('');
    const [externalName] = useState('');
    const [planId, setPlanId] = useState('');


    const handleCreate = () => {
        setLoading(true)
        axios
            .post<ServiceInstance>(api("service-instances"), 
            {name: name, service_plan_id: planId})
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
        if (!Ok(props.offering)) {
            return;
        }

        if (!Ok(props.plan)) {
            return;
        }

        setName(generateServiceInstanceName(
            props.plan?.name,
            props.offering?.catalog_name
        ))

        setPlanId(props.plan.id)

    }, [props.plan, props.offering]);

    const renderData = () => {

        return (
            <>
                <ui5.Form>
                    <ui5.FormItem label={<ui5.Label required>Name</ui5.Label>}>
                        <ui5.Input
                            style={{ width: "100%" }}
                            required
                            value={name}
                        />
                    </ui5.FormItem>
                    <ui5.FormItem label="Provisioning Parameters">
                        <ui5.TextArea
                            style={{ width: "100%", height: "100px" }}
                            valueState="None"
                            title="Provisioning Parameters"
                            value={params}
                        />
                    </ui5.FormItem>
                    <ui5.FormItem label="External Name">
                        <ui5.Input value={externalName} />
                    </ui5.FormItem>
                    <ui5.FormItem>
                        <ui5.Button  onClick={handleCreate}>Create</ui5.Button>
                    </ui5.FormItem>
                </ui5.Form>
            </>
        )
    }
    // @ts-ignore
    return <>{renderData()}</>;
}

function generateServiceInstanceName(
    plan: string | undefined,
    service: string | undefined
): string {
    const id = window.crypto.randomUUID().substring(0, 4);
    return `${service}-${plan}-${id}`;
}

export default CreateInstanceForm;