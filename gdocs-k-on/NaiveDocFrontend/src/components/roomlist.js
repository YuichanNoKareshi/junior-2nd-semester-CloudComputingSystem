import React, { useContext, useState, useEffect, useRef } from 'react';

import {
    Menu,
    Layout,
    Row,
    Col,
    Pagination,
    Input,
    Card,
    Dropdown,
    Slider,
    InputNumber
} from 'antd'

import Tooltip from "antd/es/tooltip";
import {withRouter} from "react-router-dom";
import { DownOutlined } from '@ant-design/icons'
import { STR } from './STR'
import {apiUrl} from "../constant";

// import * as RoomSER from '../services/SingleTurtleRoomService'

class RoomList extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            dataSource: [],
            count: 6,
            usersinfo: [],
            user: {},
            pagesize:20,
            pagenum:1,
            roomlist: [],

        };
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
        }
    }

    sendConnect() {
        let Message = {};
        Message.type = "connect";
        Message.username = this.state.user.username;
        // Message.username = this.state.user.username;
        let jsonStr;
        jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);
    }

    onMessage(event) {
        console.log(event.data)
        let msg = JSON.parse(event.data);
        console.log(msg)

        if (msg.type == "FileNameSlice") {
            let filenames = msg.data;
            let i = 0
            let roomlist = [];
            for (i = 0;i < filenames.length;i++) {
                let room = {};
                room.roomname = filenames[i];
                roomlist[i] = room
                console.log(room.roomname)
            }

            this.setState({ roomlist: [] })
            this.setState({ roomlist: roomlist })
        }

    }

    callback = (data) => {
        this.setState({ roomlist: [] })
        this.setState({ roomlist: data })
    }

    getRooms = () => {
        // let sortby = this.state.dropdownkey == 1 ? 0 : 1
        // let json = {
        //     pagenum: this.state.pagenum,
        //     pagesize: this.state.pagesize,
        //     sortby
        // }
        // RoomSER.getRooms(json, this.callback)
    }

    changePage = (current, pageSize) => {
        // this.setState({
        //     pagenum: current,
        //     pagesize: pageSize,
        //     roomlist: []
        // })
        // let json = { pagenum: current, pagesize: pageSize }
        // RoomSER.getRooms(json, this.callback)
    }

    renderList = () => {
        let result = []
        for (let i = 0; i < this.state.roomlist.length; i++) {
            result.push(<STR info={this.state.roomlist[i]} />)
            console.log(this.state.roomlist[i])
        }
        return result
    }


    render() {
        return (
            <Layout>
                <Layout>
                    <br />
                    <Row justify='center'>
                        <Col offset={1} span={14}>
                            <Card>
                                {this.renderList()}
                                <br />
                                <Pagination
                                    showSizeChanger
                                    showQuickJumper
                                    total={500}
                                    current={this.state.pagenum}
                                    pageSize={this.state.pagesize}
                                    onChange={this.changePage}
                                    style={{ float: 'right' }}
                                />
                            </Card>
                        </Col>
                    </Row>
                </Layout>
                <br />
                <br />
            </Layout>
        )
    }

}



export  default withRouter(RoomList);