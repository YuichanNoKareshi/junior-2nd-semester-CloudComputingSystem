import React from 'react';

import {Button, Col, Layout, Menu, Row, Modal, Space, Input} from 'antd';

import RecycleList from "../components/recyclelist";
import {StarTwoTone,BulbTwoTone, HighlightTwoTone} from "@ant-design/icons";
import {Link} from "react-router-dom";
import "../css/form.css"
import * as SheetService from '../services/SheetService'
import {apiUrl} from "../constant";
import {history} from "../utils/history";


const { Header,  Content } = Layout


class RecycleBinPage  extends React.Component{
    constructor(props) {
        super(props)
        this.state = {
            roominfo: {},
            loading: false,
            visible: false,
            user: {}
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
    }

    onMessage(event) {
        // console.log(event.data.type)

    }

    toRoomPage = () => {
        this.websocket.close()
        history.push('/Sheet/roomlist');
        window.location = '/Sheet/roomlist';
    }

    render() {
        const { visible, loading } = this.state;
        return (

            <Layout>
                <Layout>
                    <Header style={{background:"transparent"}}>
                        <Col offset={22}>
                            <>
                                <Button type="primary" onClick={this.toRoomPage}>
                                    文件列表
                                </Button>
                            </>
                        </Col>
                    </Header>
                    <Content>

                        <RecycleList/>

                    </Content>
                </Layout>
                <br />

                <br />



            </Layout>

        )

    }





}

export default RecycleBinPage;
