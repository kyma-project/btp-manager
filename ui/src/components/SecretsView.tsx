import * as ui5 from "@ui5/webcomponents-react";
import axios from "axios";
import { useEffect, useState } from "react";
import { Secrets } from "../shared/models";
import Ok from "../shared/validator";
import api from "../shared/api";
import { Button, DynamicPageTitle, Menu, ObjectStatus } from "@ui5/webcomponents-react";

function SecretsView({ onSecretChanged }: { onSecretChanged: (secret: string) => void }) {
    const [secrets, setSecrets] = useState<Secrets>();
    const [selectedSecret, setSelectedSecret] = useState("btp-operator");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [isOpen, setIsOpen] = useState(false);

    const loadSecrets = () => {
        setLoading(true);
        axios
            .get<Secrets>(api("secrets"))
            .then((response) => {
                setLoading(false);
                setSecrets(response.data);
                if (Ok(response.data) && Ok(response.data.items)) {
                    const secret = formatSecretText(response.data.items[0].name, response.data.items[0].namespace)
                    onSecretChanged(secret);
                    setSelectedSecret(secret);
                } else {
                    onSecretChanged(formatSecretText("", ""));
                }
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
                setSecrets(undefined);
                onSecretChanged(formatSecretText("", ""));
            });
            setLoading(false);
    };

    useEffect(() => {
        loadSecrets();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const fetchSecrets = () => {
        setLoading(true);
        axios
            .get<Secrets>(api("secrets"))
            .then((response) => {
                setLoading(false);
                setSecrets(response.data);
            })
            .catch((error) => {
                setLoading(false);
                setError(error);
                setSecrets(undefined);
                onSecretChanged(formatSecretText("", ""));
            });
            setLoading(false);
    };

    const renderData = () => {
        if (loading) {
            return <ui5.IllustratedMessage name="UnableToLoad" />
        }

        if (error) {
            onSecretChanged(formatSecretText("", ""));
            return <ui5.IllustratedMessage name="UnableToLoad" />
        }

        // @ts-ignore
        if (!Ok(secrets) || !Ok(secrets.items)) {
            return <ui5.MenuItem text={formatSecretText("", "")} />
        }
        return secrets?.items.map((secret, index) => {
            return (
                <ui5.MenuItem
                    text={formatSecretText(secret.name, secret.namespace)}
                    data-secret-name={secret.name}
                    data-secret-namespace={secret.namespace}
                />
            );
        });
    };

    return (
        <>
            <DynamicPageTitle actions={
                <>
                    <Button
                        design="Emphasized"
                        onClick={function _a() {
                            setIsOpen(!isOpen);
                            fetchSecrets();
                        }}
                        id="openMenu"
                    >
                    Select a secret
                    </Button>

                    <Menu
                        opener="openMenu"
                        onAfterClose={function _a() {setIsOpen(false)}}
                        onAfterOpen={function _a() { }}
                        onBeforeClose={function _a() { }}
                        onBeforeOpen={function _a() { }}
                        onItemClick={(event) => {
                            const secretName = event.detail.item.dataset.secretName;
                            const secretNamespace = event.detail.item.dataset.secretNamespace;                           
                            if (secretName && secretNamespace) {
                                setSelectedSecret(formatSecretText(secretName, secretNamespace));
                                onSecretChanged(formatSecretText(secretName, secretNamespace));
                            }
                            setIsOpen(false);
                        }}
                        onItemFocus={function _a() { }}
                        open={isOpen}
                    >
                        {renderData()}
                    </Menu>
                </>
            }

                header={selectedSecret}
                subHeader="Currently you are connected to service manager that the above secret points to. To select other environment, use `select` button on the right.">
                <ObjectStatus state="Success">connected</ObjectStatus>
            </DynamicPageTitle>
        </>
    );
}

function formatSecretText(secretName: string, secretNamespace: string) {
    if (secretName === "" || secretNamespace === "") {
        return ""
    }
    return `${secretName} in (${secretNamespace})`;
}

export default SecretsView;
