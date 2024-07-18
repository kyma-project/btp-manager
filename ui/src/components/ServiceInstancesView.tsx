import * as ui5 from "@ui5/webcomponents-react";
import { ServiceInstances } from "../shared/models";
import axios from "axios";
import { useEffect, useState, useRef } from "react";
import { createPortal } from "react-dom";
import api from "../shared/api";
import Ok from "../shared/validator";
import serviceInstancesData from '../test-data/serivce-instances.json';

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
    var useTestData = process.env.REACT_APP_USE_TEST_DATA
    if (!useTestData) {
      setLoading(true)
      axios
        .get<ServiceInstances>(api("service-instances"))
        .then((response) => {
          setLoading(false);
          setServiceInstances(response.data);
        })
        .catch((error) => {
          setLoading(false);
          setError(error);
        });
      setLoading(false)
    } else {
      setLoading(true)
      setServiceInstances(serviceInstancesData)
      setLoading(false);
    }
  }, []);

  if (loading) {
    return <ui5.Loader progress="100%" />
  }

  if (error) {
      return <ui5.IllustratedMessage name="UnableToLoad" />
  }

  const renderData = () => {
    // @ts-ignore
     if (!Ok(serviceInstances) || !Ok(serviceInstances.items)) {
        return <ui5.IllustratedMessage name="NoEntries" />
    }
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
          >
            <ui5.List>
            </ui5.List>
          </ui5.Dialog>,
          document.body
        )}
      </>
    </>
  );
}

export default ServiceInstancesView;