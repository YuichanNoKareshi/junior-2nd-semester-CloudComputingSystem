import { postRequest } from '../utils/ajax'
// import { message } from 'antd'
import { apiUrl } from '../constant'
import {history} from "../utils/history";

export const createRoom = (data) => {
    let filename = data.filename
    history.push('/Sheet/' + filename);
    window.location = '/Sheet/' + filename;
}

export const enterRoom = (data) => {
    let filename = data.filename
    history.push('/Sheet/' + filename);
    window.location = '/Sheet/' + filename;
}

