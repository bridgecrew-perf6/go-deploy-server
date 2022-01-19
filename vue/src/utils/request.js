import axios from "axios";

import { ElMessage, ElMessageBox } from 'element-plus'

//import store from "@/store";
import {
  getLocalStorage
} from "@/utils/storage";


// 切换接口
var baseApiUrl = getLocalStorage("api_url") || process.env.VUE_APP_BASE_API
if (document.location.protocol == "http:") {
  baseApiUrl = baseApiUrl.replace("https:", document.location.protocol)
}

// create an axios instance
// 设置 api_url 线上
const service = axios.create({
  baseURL: baseApiUrl, // url = base url + request url
  // withCredentials: true, // send cookies when cross-domain requests
  timeout: 20000, // request timeout
  transformRequest: [
    (data, headers) => {
      if (headers['Content-Type'] === 'application/x-www-form-urlencoded') {
        // 把一个参数对象格式化为一个字符串
        // return qs.stringify(data)
        let ret = ''
        for (const it in data) {
          ret +=
            encodeURIComponent(it) + '=' + encodeURIComponent(data[it]) + '&'
        }
        return ret.substr(0, ret.length - 1)
      } else if (headers['Content-Type'] === 'multipart/form-data;charset=UTF-8') {
        return data
      } else {
        headers['Content-Type'] = 'application/json'
      }
      return JSON.stringify(data)
    }
  ]
});

// request interceptor
service.interceptors.request.use(
  config => {
    // do something before request is sent

    let token = getLocalStorage("token");
    if (token) {
      // let each request carry token
      // ['X-Token'] is a custom headers key
      // please modify it according to the actual situation
      config.headers["Authorization"] = "Bearer " + token;
    }
    return config;
  },
  error => {
    // do something with request error
    console.log(error); // for debug
    return Promise.reject(error);
  }
);

// response interceptor
service.interceptors.response.use(
  /**
   * If you want to get http information such as headers or status
   * Please return  response => response
   */

  /**
   * Determine the request status by custom code
   * Here is just an example
   * You can also judge the status by HTTP Status Code
   */
  response => {
    const res = response.data;

    // if the custom code is not 20000, it is judged as an error.
    if (res.code !== 0) {
      ElMessage({
        message: res.message || "Error",
        type: "error",
        duration: 3 * 1000
      });

      // 50008: Illegal token; 50012: Other clients logged in; 50014: Token expired;
      if (res.code === -999) {
        // to re-login
        ElMessageBox.confirm(
          "You have been logged out, you can cancel to stay on this page, or log in again",
          "Confirm logout", {
          confirmButtonText: "Re-Login",
          cancelButtonText: "Cancel",
          type: "warning"
        }
        ).then(() => {
          //store.dispatch("user/resetToken").then(() => {
          location.reload();
          //});
        });
      }
      return Promise.reject(new Error(res.message || "Error"));
    } else {
      return res;
    }
  },
  error => {
    console.log("err" + error); // for debug
    ElMessage({
      message: error.message,
      type: "error",
      duration: 3 * 1000
    });
    return Promise.reject(error);
  }
);

export default service;