import * as ui5 from "@ui5/webcomponents-react";
import { ServiceInstance, ServiceInstances } from "../shared/models";
import axios from "axios";
import { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";
import api from "../shared/api";

function ServiceInstancesView() {
  const [serviceInstances, setServiceInstances] = useState<ServiceInstances>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const dialogRef = useRef(null);
  const handleOpen = (e: any) => {
    // @ts-ignore
    dialogRef.current.show();
  };
  const handleClose = () => {
    // @ts-ignore
    dialogRef.current.close();
  };

  useEffect(() => {
    setLoading(true)
    axios
      .get<ServiceInstances>(api("list-service-instances"))
      .then((response) => {
        setServiceInstances(response.data);
        setLoading(false);
      })
      .catch((error) => {
        setError(error);
        setLoading(false);
      });
  }, []);

  if (loading) {
    <ui5.Loader progress="60%" />
  }

  if (error) {
    return <ui5.Text>Error: {error}</ui5.Text>;
  }

  const renderData = () => {
    return serviceInstances?.items.map((brief, index) => {
      return (
        <>
          <ui5.TableRow>
            <ui5.TableCell>
              <ui5.Label>{brief.name}</ui5.Label>
            </ui5.TableCell>
            <ui5.TableCell>
              <ui5.Label>{brief.namespace}</ui5.Label>
            </ui5.TableCell>
          </ui5.TableRow>
        </>
      );
    });
  };

  return (
    <>
      <ui5.Table
        columns={
          <>
            <ui5.TableColumn>
              <ui5.Label>Service Instance</ui5.Label>
            </ui5.TableColumn>
          </>
        }
        onClick={handleOpen}
        onLoadMore={function _a() {}}
        onPopinChange={function _a() {}}
        onRowClick={function _a() { }}
        onSelectionChange={function _a() {}}
      >
        {renderData()}
      </ui5.Table>

      <>
        {createPortal(
          <ui5.Dialog
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
            header={
              <ui5.Bar endContent={<ui5.Icon name="settings" />}>
                <ui5.Title>Dialog</ui5.Title>
              </ui5.Bar>
            }
            headerText="Dialog Header"
            onAfterClose={function _a() {}}
            onAfterOpen={function _a() {}}
            onBeforeClose={function _a() {}}
            onBeforeOpen={function _a() {}}
          >
            <ui5.List>
              <ui5.StandardListItem additionalText="3">
                List Item 1
              </ui5.StandardListItem>
            </ui5.List>
          </ui5.Dialog>,
          document.body
        )}
      </>
    </>
  );
}

export default ServiceInstancesView;