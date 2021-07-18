import React from 'react'
import {
    BrowserRouter as Router,
    Route,
    Switch,
    Redirect
} from 'react-router-dom'
// import { history } from './util/history'
import Luckysheet from './components/Luckysheet'
import Roompage from "./pages/roompage";
import LoginView from "./pages/LoginView";
import RecycleBinPage from "./pages/recycleBinPage";

class BasicRoute extends React.Component {
    render() {
        return (
            <Router>
                {/* <ErrorBoundary> */}
                <Switch>
                    <Route exact path='/' component={LoginView} />
                    <Route exact path='/login' component={LoginView} />
                    <Route exact path='/Sheet/roomlist' component={Roompage} />
                    <Route exact path='/recyclebin' component={RecycleBinPage} />
                    <Route path='/Sheet/:filename' component={Luckysheet} />
                    <Redirect from='/*' to='/' />
                </Switch>
                {/* </ErrorBoundary> */}
            </Router>
        )
    }
}

export default BasicRoute