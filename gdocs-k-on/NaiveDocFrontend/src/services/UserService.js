import { postRequest } from '../utils/ajax'
import { message } from 'antd'
import { history } from '../utils/history'
import { apiUrl } from '../constant'

export const login = (data) => {
    let User = {};
    User.username = data.username;
    localStorage.setItem('user', JSON.stringify(User));
    history.push('/Sheet/roomlist');
    window.location = '/Sheet/roomlist';
}

