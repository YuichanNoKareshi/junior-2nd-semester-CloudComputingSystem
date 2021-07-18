import React from 'react'
import {Card, Tag, Row, Button, Popconfirm, Col, Modal, Space, Input, message} from 'antd'
import '../css/STR.css'
import { Link } from 'react-router-dom'
import * as SheetService from "../services/SheetService";
import {apiUrl} from "../constant";
// import * as STRService from "../services/SingleTurtleRoomService";

export class RSTR extends React.Component {
    constructor(props) {
        super(props)
        this.state = {
            roominfo: {},
            loading: false,
            visible: false,
            visible_del: false,
            user: {}
        }
        this.websocket = null;
        this.onMessage = this.onMessage.bind(this);
    }

    onMessage(event) {
        console.log(event.data)
        let msg = JSON.parse(event.data);
        if (msg.type == "RECOVERET") {
            this.websocket.close()
            window.location = '/recyclebin';
        }
        if (msg.type == "CTRET") {
            this.websocket.close()
            window.location = '/recyclebin';
        }
        if (msg.type == "MTTSUCCESS") {
            this.websocket.close()
            window.location = '/recyclebin';
        }
    }

    handleDelete = () => {
        this.setState({ loading: true });
        setTimeout(() => {
            this.setState({ loading: false, visible_del: false });
        }, 300000 );
        let filename = this.state.roominfo.roomname;

        let Message = {
            type: "ct",
            filename: filename,
            username: this.state.user.username
        }

        let jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);
    }
    handleCancel = () => {
        this.setState({ visible: false });
    };
    handleCancelDel = () => {
        this.setState({ visible_del: false });
    };
    showModal = () => {
        this.setState({
            visible: true,
        });
    };
    showModalDel = () => {
        this.setState({
            visible_del: true,
        });
    };
    handleOk = () => {
        this.setState({ loading: true });
        setTimeout(() => {
            this.setState({ loading: false, visible: false });
        }, 300000);

        let filename = this.state.roominfo.roomname;

        let Message = {
            type: "recover",
            filename: filename,
            username: this.state.user.username
        }
        console.log(Message)

        let jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);
    };
    componentDidMount() {
        let user = JSON.parse(localStorage.getItem('user'));
        this.setState({user: user});

        this.setState({ roominfo: this.props.info})
        console.log(this.props.info)
        const url = `${apiUrl}/ws`
        this.websocket = new WebSocket(url);
        this.websocket.onmessage = this.onMessage;
        this.websocket.onopen = () => {
            let Message = {
                type: "connect"
            }
            let jsonStr = JSON.stringify(Message);
            this.websocket.send(jsonStr);


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

    render() {

        const { visible, loading, visible_del } = this.state;
        return (
            <Card
                title={
                    <div>
                        <b>{this.state.roominfo.roomname}</b>
                    </div>
                }
                hoverable
                class='Task_Blower'
                extra={
                    <>
                        <Button type="primary" onClick={this.showModalDel}>
                            彻底删除
                        </Button>
                        <Modal
                            title="彻底删除"
                            visible={visible_del}



                            footer={[
                                <Button key="back" onClick={this.handleCancelDel}>
                                    返回
                                </Button>,
                                <Button key="submit" type="primary" loading={loading} onClick={this.handleDelete}>
                                    确认删除
                                </Button>,
                            ]}
                        >
                        </Modal>

                        <Button type="primary" onClick={this.showModal}>
                            恢复文件
                        </Button>
                        <Modal
                            title="恢复文件"
                            visible={visible}



                            footer={[
                                <Button key="back" onClick={this.handleCancel}>
                                    返回
                                </Button>,
                                <Button key="submit" type="primary" loading={loading} onClick={this.handleOk}>
                                    恢复文件
                                </Button>,
                            ]}
                        >
                        </Modal>
                    </>

                }
            >
            </Card>
        )
    }
}