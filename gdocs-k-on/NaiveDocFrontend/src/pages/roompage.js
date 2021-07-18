import React from 'react';

import {Button, Col, Layout, Menu, Row, Modal, Space, Input, message} from 'antd';

import RoomList from "../components/roomlist";
import {StarTwoTone,BulbTwoTone, HighlightTwoTone} from "@ant-design/icons";
import {Link} from "react-router-dom";
import "../css/form.css"
import * as SheetService from '../services/SheetService'
import {apiUrl} from "../constant";
import {history} from "../utils/history";
import {STR} from "../components/STR";


const { Header,  Content } = Layout


class Roompage  extends React.Component{
    constructor(props) {
        super(props)
        this.state = {
            roominfo: {},
            loading: false,
            visible: false,
            visibleLog: false,
            user: {},
            Log: []
        }
        this.websocket = null;
        this.onMessage = this.onMessage.bind(this);
        this.sendConnect = this.sendConnect.bind(this);
    }

    componentDidMount() {
        let user = JSON.parse(localStorage.getItem('user'));
        this.setState({user: user});

        const url = `${apiUrl}/ws`
        this.websocket = new WebSocket(url);
        this.websocket.onmessage = this.onMessage;
        this.websocket.onopen = () => {
            this.sendConnect()
        };
        this.websocket.onclose = function (e) {
            console.log('websocket 断开: ' + e.code + ' ' + e.reason + ' ' + e.wasClean)
            console.log(e)
            let Message = {
                type: "close",
                filename: ""
            }
            let jsonStr = JSON.stringify(Message);
            this.websocket.send(jsonStr);
        }
    }

    sendConnect() {
        let Message = {};
        Message.type = "connect";
        Message.username = this.state.user.username;
        console.log(Message)
        // Message.username = this.state.user.username;
        let jsonStr;
        jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);

        let MessageLog = {};
        MessageLog.type = "log";
        let jsonStrLog;
        jsonStrLog = JSON.stringify(MessageLog);
        this.websocket.send(jsonStrLog);
    }

    onMessage(event) {
        console.log(event.data)
        let msg = JSON.parse(event.data);
        if (msg.type == "CREATESUCCESS") {
            let Message = {
                type: "create",
                filename: msg.data[0]
            }

            this.websocket.close()
            SheetService.createRoom(Message)
        }

        if (msg.type == "LOG") {
            console.log(msg)
            let logs = msg.data;
            let i = 0;
            let loglist = [];
            for (i = 0;i < logs.length;i++) {
                loglist[i] = logs[i]
            }

            this.setState({Log: loglist})
        }

        if (msg.type == "CEF") {
            console.log(msg)
            this.setState({ visible: false });
            this.setState({ loading: false });
            let msgg = "不能创建同名文件！"
            message.error(msgg);
        }
    }

    toRecycleBin = () => {
        this.websocket.close()
        history.push('/recyclebin');
        window.location = '/recyclebin';
    }

    showModal = () => {
        this.setState({
            visible: true,
        });
    };
    showLog = () => {
        this.setState({
            visibleLog: true,
        });
        console.log(this.state.Log)
    }
    handleCancel = () => {
        this.setState({ visible: false });
    };
    handleCancelLog = () => {
        this.setState({ visibleLog: false });
    };
    handleOk = () => {
        this.setState({ loading: true });
        setTimeout(() => {
            this.setState({ loading: false, visible: false });
        }, 3000000 );
        var filename = document.getElementById("filename").value;

        if (filename == null || filename.search(" ") != -1 || filename.search("/") != -1 || filename.search("-") != -1 || filename.indexOf("\\") != -1) {
            this.setState({ loading: false, visible: false });
            let msgg = "不能用这些字符哟"
            message.error(msgg)
            return
        }

        let Message = {
            type: "create",
            filename: filename,
            username: this.state.user.username
        }

        let jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);

        // SheetService.createRoom(Message)
    };

    renderList = () => {
        let result = []
        for (let i = 0; i < this.state.Log.length; i++) {
            result.push(<div> {this.state.Log[i]} </div>)
        }
        return result;
    }

    render() {
        const { visible, loading, visibleLog } = this.state;
        return (

            <Layout>
                <Layout>
                    <Header style={{background:"transparent"}}>
                        <Col offset={22}>
                    <>
                        <Button type="primary" onClick={this.toRecycleBin}>
                            回收站
                        </Button>
                        <Button type="primary" onClick={this.showModal}>
                            创建文件
                        </Button>
                        <Modal
                            title="创建文件"
                            visible={visible}



                            footer={[
                                <Button key="back" onClick={this.handleCancel}>
                                    返回
                                </Button>,
                                <Button key="submit" type="primary" loading={loading} onClick={this.handleOk}>
                                    创建
                                </Button>,
                            ]}
                        >
                            <Space direction="vertical">

                                <Input id={"filename"} placeholder="input filename" />

                            </Space>
                        </Modal>
                        <Button type="primary" onClick={this.showLog}>
                            文件系统Log
                        </Button>
                        <Modal
                            title="Log"
                            visible={visibleLog}



                            footer={[
                                <Button key="back" onClick={this.handleCancelLog}>
                                    返回
                                </Button>,
                            ]}

                        >
                            {this.renderList()}
                        </Modal>
                    </>
                        </Col>
                    </Header>
                <Content>

                    <RoomList/>

                </Content>
                </Layout>
                <br />

                <br />



            </Layout>

        )

    }





}

export default Roompage;
