import * as ui5 from "@ui5/webcomponents-react";
import axios from "axios";
import { useEffect, useState } from "react";
import { Secrets } from "../shared/models";
import api from "../shared/api";

function SecretsView(props: any) {
  const [secrets, setSecrets] = useState<Secrets>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    axios
      .get<Secrets>(api("list-secrets"))
      .then((response) => {
        setSecrets(response.data);
        setLoading(false);
        props.handler(
          formatDisplay(
            response.data.items[0].name,
            response.data.items[0].namespace
          )
        );
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <ui5.Loader progress="60%" />
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  const renderData = () => {
    return secrets?.items.map((s, i) => {
      return (
        <ui5.Option key={i}>{formatDisplay(s.name, s.namespace)}</ui5.Option>
      );
    });
  };

  return (
    <div>
      <>
        <div>
          <ui5.Select
            style={{ width: "20vw" }}
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

function formatDisplay(secretName: string, secretNamespace: string) {
  return `${secretName} in (${secretNamespace})`;
}

export default SecretsView;
