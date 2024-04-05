import * as ui5 from "@ui5/webcomponents-react";
import { useEffect, useState } from "react";
import axios from "axios";
import ServiceOfferingModel from "../models/serviceOffering";

function ServiceOfferings(props: any) {
  const [offerings, setOfferings] = useState<ServiceOfferingModel>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const splited = splitSecret(props.secret);
    if (splited) {
      axios
      .get<ServiceOfferingModel>(
        `http://localhost:3002/api/list-offerings/${splited.namespace}/${splited.secretName}`
      )
      .then((response) => {
        setOfferings(response.data);
        setLoading(false);
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
    }
  }, [props.secret]);

  if (loading) {
    return <ui5.Text>Loading...</ui5.Text>;
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  const renderData = () => {
    return offerings?.items.map((offering, index) => {
      return (
        <>
          <ui5.Card
            key={index}
            header={
              <ui5.CardHeader
                avatar={<ui5.Icon />}
                subtitleText={offering.metadata.displayName}
                titleText={offering.name}
                status={formatStatus(index, offerings.num_items)}
                interactive
              />
            }
          >
            <ui5.Text>{offering.description}</ui5.Text>
          </ui5.Card>
        </>
      );
    });
  };

  return <>{renderData()}</>;
}

function splitSecret(s: string) {
  if (s == null) {
    return {};
  }

  // @ts-ignore
  const secretName = s.match(/(\w+) in/)[1];
  // @ts-ignore
  const namespace = s.match(/\((\w+)\)/)[1];

  return { secretName, namespace };
}

function formatStatus(i: number, j: number) {
  return `${++i} of ${j}`;
}

export default ServiceOfferings;