import React from 'react'
import { Form, Input, Button, Checkbox } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { Link } from 'react-router-dom'
import * as UserSER from '../services/UserService'


const formItemLayout = {
    labelCol: {
        xs: { span: 16 },
        sm: { span: 8 }
    },
    wrapperCol: {
        xs: { span: 24 },
        sm: { span: 16 }
    }
}

const tailFormItemLayout = {
    wrapperCol: {
        xs: {
            span: 24,
            offset: 0
        },
        sm: {
            span: 16,
            offset: 8
        }
    }
}

class Loginform extends React.Component {

    onSubmit = (values) => {
        // debugger
        // console.log('Received values of form: ', values);
        // console.log(values);
        // let User = {};
        // User.username = values.username;
        // localStorage.setItem('user', JSON.stringify(User));
        // let user = JSON.parse(localStorage.getItem('user'));
        // console.log(user.username);
        UserSER.login(values)
    }

    render() {
        //  debugger;
        return (
            <Form
                {...formItemLayout}
                name='normal_login'
                // initialValues={{ remember: true }}
                onFinish={this.onSubmit}
            >
                <Form.Item
                    label='使用此用户名进入NaiveDoc'
                    name='username'
                    rules={[{ required: true, message: '请输入你的用户名' }]}
                >
                    <Input
                        id="username"
                        prefix={<UserOutlined className='site-form-item-icon' />}
                        placeholder='Username'
                    />
                </Form.Item>

                <Form.Item {...tailFormItemLayout}>
                    <Button type='primary' htmlType='submit' className='form-button'>
                        进入NaiveDoc！
                    </Button>
                </Form.Item>
            </Form>
        )
    }
}

export default Loginform