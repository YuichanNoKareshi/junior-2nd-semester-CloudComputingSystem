import React, {Component} from 'react';
import {Input, Button ,notification} from 'antd';
import 'antd/dist/antd.css';
import '../css/login.css'
import { UserOutlined } from '@ant-design/icons'
import * as UserSER from "../services/UserService";

class LoginView extends Component {
    constructor(props) {
        super(props);
        this.state ={
            username:'',
            password:'',
        }
        this.userChange = this.userChange.bind(this);
        this.passwordChange = this.passwordChange.bind(this);
        this.handleSubmit = this.handleSubmit.bind(this);
        this.handleSignUp = this.handleSignUp.bind(this);
        this.callback = this.callback.bind(this);
    }

    userChange(e){
        this.setState({ username : e.target.value })
    };

    passwordChange(e){
            this.setState({ password : e.target.value })
    };

    handleSubmit = () => {
        if (this.state.username==='')
        {
            this.alert('ERROR!','Please input your username!');
            return;
        }

        let values = {
            username: this.state.username
        }

        UserSER.login(values)

    };

    callback = (data) => {
        if(data != null) {
            if(data.ban === 0)
            {
                sessionStorage.setItem('user', data.username);
                sessionStorage.setItem('isAdmin', data.administrator);
                if (data.administrator===0)
                    this.props.history.replace({pathname: '/home'});
                else
                    this.props.history.replace({pathname: '/ADhome'});
            }
            else
                this.alert('WARNING!','You are forbidden to login our ebook!');
        }
    };


    handleSignUp = () => {
        this.props.history.replace({pathname: '/signup'});
    };

    alert = (mess,des) =>{
        notification.open({
            message: mess,
            description: des,
        });
    };

    render() {
        return (
                <div className="login-page">
                    <div className="login-container">
                        <div className="login-box">
                            <h1 className="page-title">Login</h1>
                            <div className="login-content">

                                <Input size="large" placeholder="username" prefix={<UserOutlined />} onChange={this.userChange}/>
                                <br/>
                                <br/>
                                <Button shape="round" size="large"  onClick={this.handleSignUp} >sign up</Button>
                                <Button type="primary" shape="round" size="large"  style={{float:"right"}} onClick={this.handleSubmit}>login</Button>

                            </div>
                        </div>
                    </div>

                </div>
        );


    }
}


export default LoginView;
