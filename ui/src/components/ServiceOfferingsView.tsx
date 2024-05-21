import * as ui5 from "@ui5/webcomponents-react";
import { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";

import axios from "axios";
import { ServiceOfferingDetails, ServiceOfferings } from "../shared/models";
import ts from "typescript";
import api from "../shared/api";

function ServiceOfferingsView(props: any) {
  const [offerings, setOfferings] = useState<ServiceOfferings>();
  const [serviceOfferingDetails, setServiceOfferingDetails] =
    useState<ServiceOfferingDetails>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [labels, setLabels] = useState<JSX.Element>();
  const dialogRef = useRef(null);
  const handleOpen = (id: any) => {
    // @ts-ignore
    dialogRef.current.show();
    load(id);
  };
  const handleClose = () => {
    // @ts-ignore
    dialogRef.current.close();
  };

  useEffect(() => {
    const splited = splitSecret(props.secret);
    if (splited) {
      setLoading(true);
      axios
        .get<ServiceOfferings>(
          api(
            `list-service-offerings/${splited.namespace}/${splited.secretName}`
          )
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
  }, []);

  if (loading) {
    return <ui5.Text>Loading...</ui5.Text>;
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  function getImg(b64: string) {
    if (b64 == null) {
      return "";
    } else {
      return b64;
    }
  }

  function load(id: string) {
    setLoading(true);
    axios
      .get<ServiceOfferingDetails>(api(`get-service-offering/${id}`))
      .then((response) => {
        setServiceOfferingDetails(response.data);
        setLoading(false);
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
  }

  const renderData = () => {
    return offerings?.items.map((offering, index) => {
      return (
        <>
          <ui5.Card
            key={index}
            style={{
              width: "30%",
              height: "10%",
              padding: "1rem",
            }}
            onClick={() => {
              handleOpen(offering.id);
            }}
            header={
              <ui5.CardHeader
                avatar={
                  <ui5.Avatar>
                    <img alt="" src={getImg(offering.metadata.imageUrl)}></img>
                  </ui5.Avatar>
                }
                subtitleText={offering.metadata.displayName}
                titleText={offering.catalogName}
                status={formatStatus(index, offerings?.numItems)}
                interactive
              />
            }
          ></ui5.Card>

          <>
            {createPortal(
              <ui5.Dialog
                style={{ width: "800px" }}
                ref={dialogRef}
                className="headerPartNoPadding footerPartNoPadding"
                footer={
                  <ui5.Bar
                    design="Footer"
                    endContent={
                      <ui5.Button onClick={handleClose}>Close</ui5.Button>
                    }
                  />
                }
                onAfterClose={function _a() {}}
                onAfterOpen={function _a() {}}
                onBeforeClose={function _a() {}}
                onBeforeOpen={function _a() {}}
              >
                <ui5.Text>{serviceOfferingDetails?.longDescription}</ui5.Text>

                <ui5.Panel
                  accessibleRole="Form"
                  headerLevel="H2"
                  headerText="Create Service Instance"
                  onToggle={function _a() {}}
                >
                  <ui5.Form>
                    <ui5.FormItem label="Name">
                      <ui5.Input></ui5.Input>
                    </ui5.FormItem>
                    <ui5.FormItem label="External Name">
                      <ui5.Input></ui5.Input>
                    </ui5.FormItem>
                    <ui5.FormItem label="Provisioning Parameters">
                      <ui5.TextArea
                        style={{ width: "100%" }}
                        onChange={function _a() {}}
                        onInput={function _a() {}}
                        onScroll={function _a() {}}
                        onSelect={function _a() {}}
                        valueState="None"
                        title="Provisioning Parameters"
                      />
                    </ui5.FormItem>
                    <ui5.FormItem label="Plan Name">
                      <ui5.Select
                        onChange={function _a() {}}
                        onClose={function _a() {}}
                        onLiveChange={function _a() {}}
                        onOpen={function _a() {}}
                        valueState="None"
                      >
                        <ui5.Option>Option 1</ui5.Option>
                      </ui5.Select>
                    </ui5.FormItem>
                    <ui5.Icon
                      name="add"
                      onClick={() => {
                        const it = (
                          <ui5.FormItem>
                            <ui5.Input style={{ width: "50px" }}></ui5.Input>
                            <ui5.Input style={{ width: "50px" }}></ui5.Input>
                          </ui5.FormItem>
                        );
                        // @ts-ignore
                        setLabels([...labels, it]);
                      }}
                    />
                    <ui5.FormItem label="Labels">{labels}</ui5.FormItem>
                    <ui5.FormItem label="Annotations">
                      <ui5.Icon name="add" />
                      <ui5.FormItem>
                        <ui5.Input style={{ width: "50px" }}></ui5.Input>
                      </ui5.FormItem>
                      <ui5.FormItem>
                        <ui5.Input style={{ width: "50px" }}></ui5.Input>
                      </ui5.FormItem>
                    </ui5.FormItem>
                    <ui5.FormItem>
                      <ui5.Button
                        style={{ width: "100px" }}
                        onClick={function _a() {}}
                      >
                        Create
                      </ui5.Button>
                    </ui5.FormItem>
                  </ui5.Form>
                </ui5.Panel>
              </ui5.Dialog>,
              document.body
            )}
          </>
        </>
      );
    });
  };

  return <>{renderData()}</>;
}

function splitSecret(secret: string) {
  if (secret == null) {
    return {};
  }
  const secretParts = secret.split(" ");
  const secretName = secretParts[0];
  let namespace = secretParts[2].replace("(", "");
  namespace = namespace.replace(")", "");
  return { secretName, namespace };
}

function formatStatus(i: number, j: number) {
  return `${++i} of ${j}`;
}

export default ServiceOfferingsView;
