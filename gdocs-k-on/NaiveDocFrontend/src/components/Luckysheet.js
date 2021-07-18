import React from 'react';
import { withRouter, Link } from 'react-router-dom'
import { Layout, Menu, Breadcrumb ,Col,Row,Button} from 'antd';
import {  StarTwoTone, BulbTwoTone,HighlightTwoTone} from '@ant-design/icons';
import { apiUrl } from "../constant";
import {message} from 'antd';
import * as SheetServices from "../services/SheetService"

const { SubMenu } = Menu;
const { Header, Content, Sider } = Layout;
class Luckysheet extends React.Component {

    constructor(props) {
        super(props);
        this.websocket = null;
        this.state = {
            updateLock: [],
            editLock: [],
            message:"",
            user: {},
            filename: "",
            isEditing: 0,
            editingRow: -1,
            editingColumn: -1,
            isLoading: 0
        };
        this.onMessage = this.onMessage.bind(this);
        // this.clickMessage = this.clickMessage.bind(this);
        // this.onChange = this.onChange.bind(this);
        this.joinDoc = this.joinDoc.bind(this);
    }

    componentDidMount() {
        let user = JSON.parse(localStorage.getItem('user'));
        this.setState({user: user});
        let filename = this.props.match.params.filename;
        this.setState({filename: filename});

        let i = 0;
        let j = 0;
        let updateLock = [];
        let editLock = [];
        for (i = 0;i < 84;i++) {
            for (j = 0;j < 60;j++) {
                updateLock[60 * i + j] = 0;
                editLock[60 * i + j] = 0;
            }
        }

        this.setState({updateLock: updateLock})
        this.setState({editLock: editLock})

        const url = `${apiUrl}/ws`
        this.websocket = new WebSocket(url);
        this.websocket.onmessage = this.onMessage;
        this.websocket.onopen = () => {
            this.joinDoc()
        };
        this.websocket.onclose = function (e) {
            console.log('websocket 断开: ' + e.code + ' ' + e.reason + ' ' + e.wasClean)
            console.log(e)
            let Message = {};
            Message.type = "close";
            Message.username = this.state.user.username;
            Message.filename = this.state.filename;
            let jsonStr;
            jsonStr = JSON.stringify(Message);
            this.websocket.send(jsonStr);
        }

        const luckysheet = window.luckysheet;
        let that = this;
        luckysheet.create({
            container: "luckysheet",
            plugins:['chart'],
            showtoolbar: false,
            showinfobar: true,
            showsheetbar: false,
            showstatisticBar: false,
            myFolderUrl: "roomlist",
            title: that.props.match.params.filename,
            hook:{
                cellUpdated:function(r,c,oldValue, newValue, isRefresh) {
                    // finish writing a cell and update
                    // console.log(oldValue)
                    // console.log(newValue)
                    if (newValue != null) {
                        if (oldValue != newValue.v) {
                            that.updateCell(r, c, newValue.v);
                        }
                    }
                },
                cellEditBefore:function(range ) {
                    // this user is editing this cell
                    that.editingCell(range[0].row[0], range[0].column[0]);
                },
                cellUpdateBefore:function(r,c,value,isRefresh){
                    // console.info('cellUpdateBefore',r,c,value,isRefresh)
                },
                cellMousedownBefore:function(cell,postion,sheetFile,ctx){
                    console.log(postion);
                    that.MouseDownBefore(postion.r, postion.c);
                },
            }
        });

        window.addEventListener("keydown", this.handleKeyPress);
    }

    handleKeyPress = (e) => {
        console.log(e.keyCode)
        if(e.keyCode===17){
            let Message = {};
            Message.type = "rollback";
            Message.username = this.state.user.username;
            Message.filename = this.state.filename;
            let jsonStr;
            jsonStr = JSON.stringify(Message);
            this.websocket.send(jsonStr);
            console.log(Message)
        }
    };

    MouseDownBefore(r, c) {
        const luckysheet = window.luckysheet;
        if (this.state.isEditing == 1) {
            this.setState({isEditing: 0})
            let newValue = luckysheet.getCellValue(this.state.editingRow, this.state.editingColumn);
            this.updateCell(this.state.editingRow, this.state.editingColumn, newValue);
        }
    }

    editingCell(r, c) {
        const luckysheet = window.luckysheet;
        let editLock = this.state.editLock;
        let updateLock = this.state.updateLock;

        if (this.state.isEditing == 1) {
            let newValue = luckysheet.getCellValue(this.state.editingRow, this.state.editingColumn);
            this.updateCell(this.state.editingRow, this.state.editingColumn, newValue);
            this.setState({editingRow: r})
            this.setState({editingColumn: c})
        }

        if (editLock[60 * r + c] == this.state.user.username || editLock[60 * r + c] == 0) {
            this.setState({isEditing: 1})
            this.setState({editingRow: r})
            this.setState({editingColumn: c})

            let Message = {};
            Message.type = "editing";
            Message.row = r;
            Message.column = c;
            Message.username = this.state.user.username;
            Message.filename = this.state.filename;
            let jsonStr;
            jsonStr = JSON.stringify(Message);
            this.websocket.send(jsonStr);
        }
        else {
            let msg = editLock[60 * r + c] + " is editing now!"
            message.error(msg);
        }
    }

    updateCell(r, c, newValue) {
        let updateLock = this.state.updateLock;
        let editLock = this.state.editLock;
        if (updateLock[60 * r + c] == 1 || this.state.isLoading == 1) {
            console.log(newValue)
            updateLock[60 * r + c] = 0;
            this.setState({updateLock: updateLock})
        }
        else {
            if (editLock[60 * r + c] == this.state.user.username) {
                // actually we still need to fix the bug here, one can not update when lock is 0
                let Message = {};
                Message.type = "update";
                Message.row = r;
                Message.column = c;
                if (newValue == null) newValue = "";
                Message.newValue = newValue + "";
                Message.filename = this.state.filename;
                Message.username = this.state.user.username;
                console.log(Message)
                let jsonStr;
                jsonStr = JSON.stringify(Message);
                this.websocket.send(jsonStr);

                editLock[60 * r + c] = 0;
                this.setState({editLock: editLock})
                this.setState({isEditing: 0})
                this.setState({editingRow: -1})
                this.setState({editingColumn: -1})
            }
            else {
                if (editLock[60 * r + c] != 0) {
                    let msg = editLock[60 * r + c] + " is editing now!"
                    message.error(msg);
                }
            }
        }
    }

    joinDoc() {
        let Message = {};
        Message.type = "open";
        Message.filename = this.props.match.params.filename;
        Message.username = this.state.user.username;
        let jsonStr;
        jsonStr = JSON.stringify(Message);
        this.websocket.send(jsonStr);
    }

    async setCellValue(r, c, v) {
        // let updateLock = this.state.updateLock;
        // updateLock[60 * row + column] = 1;
        // this.setState({updateLock: updateLock})
        // luckysheet.setCellValue(row, column, newValue);
    }

    onMessage(event){
        console.log(event.data);
        const luckysheet = window.luckysheet;

        let msg = JSON.parse(event.data);
        if (msg.type == "update") {
            let row = msg.row;
            let column = msg.column;
            let newValue = msg.newValue;
            console.log(row, column, newValue);

            let updateLock = this.state.updateLock;
            let editLock = this.state.editLock;
            updateLock[60 * row + column] = 1;
            this.setState({updateLock: updateLock})
            luckysheet.setCellValue(row, column, newValue);

            editLock[60 * row + column] = 0;
            this.setState({editLock: editLock})
        }

        if (msg.type == "editing") {
            let row = msg.row;
            let column = msg.column;
            let username = msg.username;

            let editLock = this.state.editLock;
            editLock[60 * row + column] = username;


            if (username != this.state.user.username) {
                let editingMessage = username + " is editing";
                let updateLock = this.state.updateLock;
                updateLock[60 * row + column] = 1;
                this.setState({updateLock: updateLock})
                luckysheet.setCellValue(row, column, editingMessage);
            }
        }

        if (msg.type == "FAILOCK") {
            let row = msg.row;
            let column = msg.column;
            let username = msg.successUsername;
            let rejectUsername = msg.rejectUsername;

            if (rejectUsername == this.state.username) {
                let editLock = this.state.editLock;
                editLock[60 * row + column] = username;


                if (username != this.state.user.username) {
                    let editingMessage = username + " is editing";
                    let updateLock = this.state.updateLock;
                    updateLock[60 * row + column] = 1;
                    this.setState({updateLock: updateLock})
                    luckysheet.setCellValue(row, column, editingMessage);
                    message.error(editingMessage);
                }
            }
        }

        if (msg.type == "FileDataSlice") {
            this.setState({isLoading: 1})
            console.log(msg.data)
            let i = 0;
            let j = 0;
            for (i = 0;i < 84;i++) {
                for (j = 0;j < 60;j++) {
                    if (msg.data[60 * i + j] != null) {
                        // let updateLock = this.state.updateLock;
                        // updateLock[60 * i + j] = 1;
                        // this.setState({updateLock: updateLock})
                        luckysheet.setCellValue(i, j, msg.data[60 * i + j]);
                    }
                }
            }
            this.setState({isLoading: 0})

        }

        if (msg.type == "FILELOCKINFO") {
            let filelockinfo = msg.data;
            let i = 0;
            for (i = 0;i < filelockinfo.length;i++) {
                let row = filelockinfo[i].row;
                let column = filelockinfo[i].column;
                let username = filelockinfo[i].owner;

                let editLock = this.state.editLock;
                editLock[60 * row + column] = username;


                if (username != this.state.user.username) {
                    let editingMessage = username + " is editing";
                    let updateLock = this.state.updateLock;
                    updateLock[60 * row + column] = 1;
                    this.setState({updateLock: updateLock})
                    luckysheet.setCellValue(row, column, editingMessage);
                }
            }
        }

        if (msg.type == "ROLLBACK") {
            let row = msg.row;
            let column = msg.column;
            let oldValue = msg.oldValue;

            let updateLock = this.state.updateLock;
            let editLock = this.state.editLock;
            updateLock[60 * row + column] = 1;
            this.setState({updateLock: updateLock})
            luckysheet.setCellValue(row, column, oldValue);

            editLock[60 * row + column] = 0;
            this.setState({editLock: editLock})
        }

        if (msg.type == "ROLLBACKEMPTY") {
            let msg = "There is nothing to Rollback!"
            message.error(msg);
        }

        if (msg.type == "ROLLBACKERR") {
            let msg = "别人写了你就不能rollback了"
            message.error(msg);
        }

        if (msg.type == "ROLLBACKLOCK") {
            let msg = "别人在写不能rollback"
            message.error(msg);
        }

    }

    render() {
        const luckyCss = {
            margin: '0px',
            padding: '0px',
            position: 'absolute',
            width: '100%',
            height: '100%',
            left: '0px',
            top: '0px'
        }
        return (
            <Layout>
                <div>test</div>


                <div
                    id="luckysheet"
                    style={luckyCss}
                ></div>
            </Layout>

        )
    }
}

export default Luckysheet
