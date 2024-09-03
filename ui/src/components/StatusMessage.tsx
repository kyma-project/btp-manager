import * as ui5 from "@ui5/webcomponents-react";

import '@ui5/webcomponents/dist/features/InputElementsFormSupport.js';
import { ApiError } from "../shared/models";
import { useEffect, useState } from "react";

interface StatusMessageProps {
    error: ApiError | undefined;
    success: string | undefined;
}

function StatusMessage(props: StatusMessageProps) {

    const [message, setMessage] = useState("");


    useEffect(() => {

        if (props.error) {
            if ("name" in props.error && "code" in props.error && "message" in props.error) {
                var message = props.error!!.name + " - " + props.error!!.code + " - " + props.error!!.message;

                if (props.error!!.response) {
                    message += " - " + props.error!!.response.data;
                    setMessage(message);
                }

            } else {
                console.log(props.error);
            }
        } else if (props.success) {
            setMessage(props.success);
        }
    }, [props.error, props.success]);


    const renderData = () => {

        if (props.error && message) {
            return (
                <ui5.MessageStrip
                    design="Negative"
                    onClose={function _s() {
                        setMessage("");
                    }}>
                    {message}
                </ui5.MessageStrip>
            );
        } else if (props.success && message) {
            return (
                <ui5.MessageStrip
                    design="Information"
                    onClose={function _s() {
                        setMessage("");
                    }}>
                    {message}
                </ui5.MessageStrip>
            );
        } else {
            <div></div>
        }
    };

    return <>{renderData()}</>;
}

export default StatusMessage;