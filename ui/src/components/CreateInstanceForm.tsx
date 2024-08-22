import * as ui5 from "@ui5/webcomponents-react";
import Ok from "../shared/validator";
import {
    ApiError,
    CreateServiceInstance,
} from "../shared/models";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import api from "../shared/api";

import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';
import StatusMessage from "./StatusMessage";
import { MultiInput } from "@ui5/webcomponents-react";

function CreateInstanceForm(props: any) {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [loading, setLoading] = useState(true);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [error, setError] = useState<ApiError>();
    const [success, setSuccess] = useState("");

    const [createServiceInstance, setCreateServiceInstance] = useState<CreateServiceInstance>();
    const [labels, setLabels] = useState<string[]>([]);

    const navigate = useNavigate();

    const handleCreate = (e: any): boolean => {
        if (e.nativeEvent.submitter.localName === "ui5-multi-input") {
            e.preventDefault()
            return false;
        }

        console.log(e);
        setLoading(true)
        console.log(createServiceInstance)

        var createdJson = {
            id: createServiceInstance?.id,
            name: createServiceInstance?.name, 
            service_plan_id: createServiceInstance?.service_plan_id, 
            labels: createServiceInstance?.labels, 
            parameters: {}
        }

        if(createServiceInstance?.parameters !== undefined) {
            createdJson.parameters = JSON.parse(createServiceInstance?.parameters)
        }

        axios
            .post<CreateServiceInstance>(api("service-instances"), createdJson)
            .then((response) => {
                setLoading(false);
                setSuccess("Item with id " + response.data.name + " created, redirecting to instances page...");
                setCreateServiceInstance(new CreateServiceInstance());

                setTimeout(() => {
                    navigate("/instances/" + response.data.id);
                }, 2000);
            })
            .catch((error: ApiError) => {
                setLoading(false);
                setError(error);
            });
        e.preventDefault();
        e.stopPropagation();
        return false;
    }

    useEffect(() => {
        if (!Ok(props.offering)) {
            return;
        }

        if (!Ok(props.plan)) {
            return;
        }

        var createServiceInstance = new CreateServiceInstance();
        createServiceInstance.name = generateServiceInstanceName(
            props.plan?.name,
            props.offering?.catalog_name
        )

        createServiceInstance.service_plan_id = props.plan.id

        setCreateServiceInstance(createServiceInstance);

    }, [props.plan, props.offering]);

    function refresh(addedValue: string[]) {
        var allLabels = [...labels, ...addedValue]

        createServiceInstance!!.labels = {}
        allLabels.forEach(label => {
            var splitted = label.split("=")
            var key = splitted[0]
            var value = label.replace(key + "=", "")

            var existing = createServiceInstance!!.labels[key];
            if (existing) {
                createServiceInstance!!.labels[key] = [...existing, value];
            } else {
                createServiceInstance!!.labels[key] = [value];
            }
        });
        setLabels(allLabels);
        setCreateServiceInstance(createServiceInstance);
    }


    const renderData = () => {

        return (
            <>
                <ui5.Form
                    onSubmit={handleCreate}>
                    <ui5.FormItem>
                        <StatusMessage error={error ?? undefined} success={success} />
                    </ui5.FormItem>
                    <ui5.FormItem label={<ui5.Label required>Name</ui5.Label>}>
                        <ui5.Input
                            style={{ width: "100%" }}
                            required
                            value={createServiceInstance?.name ?? ''}
                            onChange={(e) => {
                                createServiceInstance!!.name = e.target.value
                                setCreateServiceInstance(createServiceInstance)
                            }}
                        />
                    </ui5.FormItem>

                    <ui5.FormItem label="Labels">
                        <MultiInput
                            onSubmit={function _a(e) {
                                e.preventDefault();
                            }}
                            onChange={function _a(e) {
                                var addedValue = e.target.value
                                refresh([addedValue]);

                                console.log(createServiceInstance);
                                console.log(labels);
                                e.target.value = "";
                            }}
                            onTokenDelete={function _a(e) {
                                var index = Array.prototype.indexOf.call(e.detail.token.parentElement?.children, e.detail.token);
                                console.log(index);
                                labels.splice(index, 1);
                                refresh([]);
                            }}
                            style={{ width: "100%" }}
                            tokens={labels.map(label => <ui5.Token text={label} />)}
                            type="Text"
                            valueState="None"
                            placeholder='Enter a label with a "key=value" format. After adding a label, press "Enter" to add another label'
                        />
                    </ui5.FormItem>

                    <ui5.FormItem label="Provisioning Parameters">
                        <ui5.TextArea
                            style={{ width: "100%", height: "100px" }}
                            valueState="None"
                            title="Provisioning Parameters"
                            value={createServiceInstance?.parameters ?? ''}
                            onChange={(e) => {
                                createServiceInstance!!.parameters = e.target.value
                                setCreateServiceInstance(createServiceInstance)
                            }}
                        />
                    </ui5.FormItem>

                    <ui5.FormItem>
                        <ui5.Button type={ui5.ButtonType.Submit}>Submit</ui5.Button>
                    </ui5.FormItem>

                </ui5.Form>
            </>
        )
    }
    // @ts-ignore
    return renderData();
}

function generateServiceInstanceName(
    plan: string | undefined,
    service: string | undefined
): string {
    const id = window.crypto.randomUUID().substring(0, 4);
    return `${service}-${plan}-${id}`;
}

export default CreateInstanceForm;