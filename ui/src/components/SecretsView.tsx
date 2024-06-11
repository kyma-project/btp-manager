import * as ui5 from "@ui5/webcomponents-react";
import axios from "axios";
import {useEffect, useState} from "react";
import {Secrets} from "../shared/models";
import Ok from "../shared/validator";
import api from "../shared/api";

function SecretsView(props: any) {
    const [secrets, setSecrets] = useState<Secrets>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<Secrets>(api("secrets"))
            .then((response) => {
                setLoading(false);
                setSecrets(response.data);
                if (Ok(response.data) && Ok(response.data.items)) {
                    props.handler(formatSecretText(response.data.items[0].name, response.data.items[0].namespace));
                } else {
                    props.handler(formatSecretText("", ""));
                }
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
                setSecrets(undefined);
                props.handler(formatSecretText("", ""));
            });
        setLoading(false);
    }, []);

    if (loading) {
        return <ui5.Loader progress="100%"/>
    }

    if (error) {
        props.handler(formatSecretText("", ""));
        return <ui5.IllustratedMessage name="UnableToLoad" style={{height: "50vh", width: "30vw"}}/>
    }

    const renderData = () => {
        // @ts-ignore
        if (!Ok(secrets) || !Ok(secrets.items)){
            return <div>
                <>
                    <ui5.Option key={0}>{formatSecretText("", "")}</ui5.Option>
                </>
            </div>
        } 
        return secrets?.items.map((secret, index) => {
            return (
                <ui5.Option key={index}>{formatSecretText(secret.name, secret.namespace)}</ui5.Option>
            );
        });
    };

    return (
        <div>
            <>
                <div>
                    <ui5.Select
                        style={{width: "20vw"}}
                        onChange={(e) => {
                            // @ts-ignore
                            props.handler(e.target.value);
                        }}
                    >
                        {renderData()}
                    </ui5.Select>
                </div>
            </>
        </div>
    );
}

function formatSecretText(secretName: string, secretNamespace: string) {
    if (secretName === "" || secretNamespace === "") {
        return "No secret found"
    }
    return `${secretName} in (${secretNamespace})`;
}

export default SecretsView;
