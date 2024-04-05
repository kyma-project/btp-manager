import * as ui5 from "@ui5/webcomponents-react";
import axios from "axios";
import { FormEventHandler, useEffect, useState} from "react";
import k8sSecretModel from "../models/k8sSecret";

function Secrets(props: any) {
  const [secrets, setSecrets] = useState<k8sSecretModel>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    axios
      .get<k8sSecretModel>("http://localhost:3002/api/listSecrets")
      .then((response) => {
        setSecrets(response.data);
        console.log(response.data);
        setLoading(false);
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <ui5.Text>Loading...</ui5.Text>;
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  const renderData = () => {
    return secrets?.items.map((s, i) => {
      return (
        <ui5.Option key={i}>
          {formatDisplay(s.name, s.namespace)}
        </ui5.Option>
      );
    });
  };

  return (
      <div>
        <>
          <div>
            <ui5.Select style={{width: '20vw'}} onChange={(e) => {
              // @ts-ignore
              props.handler(e.target.value);
            }}>
              {renderData()}
            </ui5.Select>
          </div>
        </>
      </div>
  );
}

function formatDisplay(secretName: string, secretNamespace: string) {
  return `${secretName} in (${secretNamespace})`;
}

export default Secrets;