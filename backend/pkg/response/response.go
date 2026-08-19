package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

const (
	CodeSuccess        = 1000
	CodeError          = 2000
	CodeInvalidParam   = 2001
	CodeUnauthorized   = 4010
	CodeLicenseInvalid = 4030
)

/*
统一输出 API 响应体，是成功、失败等快捷响应函数的底层封装。
它负责把业务状态码、提示信息和响应数据组装成固定 JSON 结构，
并通过 gin.Context 使用指定的 HTTP 状态码返回给客户端。
*/
func Result(c *gin.Context, httpStatus int, code int, msg string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: data,
	})
}

/*
返回标准成功响应。
业务接口执行成功时调用该函数，它固定使用 HTTP 200 和 CodeSuccess，
并把传入的 data 作为响应数据返回给前端。
*/
func Success(c *gin.Context, data interface{}) {
	Result(c, http.StatusOK, CodeSuccess, "success", data)
}

/*
返回标准业务失败响应。
该函数固定使用 HTTP 200，但响应体中的 code 和 msg 表示具体业务错误，
适用于参数错误、业务校验失败等不需要改变 HTTP 状态码的场景。
*/
func Fail(c *gin.Context, code int, msg string) {
	Result(c, http.StatusOK, code, msg, nil)
}

/*
返回可指定 HTTP 状态码的失败响应。
当接口需要同时表达 HTTP 层错误和业务错误时调用该函数，
例如未授权、服务异常等需要非 200 HTTP 状态码的场景。
*/
func FailWithStatus(c *gin.Context, httpStatus int, code int, msg string) {
	Result(c, httpStatus, code, msg, nil)
}

/*
返回携带额外数据的业务失败响应。
该函数固定使用 HTTP 200，并允许在失败响应中附带 data，
适用于前端需要根据错误详情继续展示或处理的场景。
*/
func FailWithData(c *gin.Context, code int, msg string, data interface{}) {
	Result(c, http.StatusOK, code, msg, data)
}
