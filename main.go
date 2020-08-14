package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fastwego/wxpay/apis/corp_pay"
	"github.com/fastwego/wxpay/apis/dev_util"
	"github.com/fastwego/wxpay/apis/download"
	"github.com/fastwego/wxpay/apis/native"
	"github.com/fastwego/wxpay/apis/order"
	"github.com/fastwego/wxpay/apis/profit_sharing"
	"github.com/fastwego/wxpay/apis/refund"
	"github.com/fastwego/wxpay/util"

	"github.com/fastwego/wxpay"
	"github.com/fastwego/wxpay/types"

	"github.com/spf13/viper"

	"github.com/gin-gonic/gin"
)

var pay *wxpay.WXPay

func Init() {
	// 加载配置文件
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	pay = wxpay.New(wxpay.Config{
		Appid:  viper.GetString("APPID"),
		Mchid:  viper.GetString("MCHID"),
		ApiKey: viper.GetString("APIKEY"),
		//IsSandboxMode: true,
		Cert: viper.GetString("CERT"),
	})

	// 开启沙箱模式
	if pay.Config.IsSandboxMode {
		params := map[string]string{
			"mch_id":    pay.Config.Mchid,
			"nonce_str": util.GetRandString(32),
		}
		result, err := dev_util.GetSignKey(pay, params)
		if err != nil {
			panic(err.Error())
		}
		pay.Config.ApiKey = result["sandbox_signkey"]
	}
}

func main() {

	Init()

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 订单通知回调
	router.POST("/api/weixin/paymentnotify", func(c *gin.Context) {
		paymentNotifyParams, err := pay.Server.PaymentNotify(c.Request)
		fmt.Println(paymentNotifyParams, err)

		// 响应接收
		err = pay.Server.ResponseSuccess(c.Writer, c.Request)
		if err != nil {
			fmt.Println(err)
		}

		// TODO 处理订单逻辑

	})

	// 退款通知回调
	router.POST("/api/weixin/refundnotify", func(c *gin.Context) {
		refundNotifyParams, err := pay.Server.RefundNotify(c.Request)
		fmt.Println(refundNotifyParams, err)

		// 响应接收
		err = pay.Server.ResponseSuccess(c.Writer, c.Request)
		if err != nil {
			fmt.Println(err)
		}

		// TODO 处理退款逻辑

	})

	// 统一下单
	router.GET("/api/wxpay/unifiedorder", func(c *gin.Context) {
		// 沙箱测试用例 https://mp.weixin.qq.com/s/W1JvZJkxTaNKm0vDw06jUw

		params := map[string]string{
			"appid":            pay.Config.Appid,
			"mch_id":           pay.Config.Mchid,
			"nonce_str":        util.GetRandString(32),
			"body":             "BODY",
			"out_trade_no":     "NO.10086",
			"total_fee":        c.Request.URL.Query().Get("fee"), // 201
			"spbill_create_ip": "12.123.14.223",
			"notify_url":       viper.GetString("NOTIFYURL"),
			"trade_type":       types.TradeTypeAPP,
		}
		result, err := order.UnifiedOrder(pay, params)
		fmt.Println(result, err)

		if err != nil {
			return
		}

		// 返回客户端预下单信息
		//result["prepay_id"]
	})

	// 查询订单
	router.GET("/api/wxpay/orderquery", func(c *gin.Context) {

		result, err := order.OrderQuery(pay, map[string]string{
			"appid":        pay.Config.Appid,
			"mch_id":       pay.Config.Mchid,
			"nonce_str":    util.GetRandString(32),
			"out_trade_no": "NO.10086",
		})

		fmt.Println(result, err)

		if err != nil {
			return
		}

		c.Writer.WriteString(fmt.Sprint(result))
	})

	// 关闭订单
	router.GET("/api/wxpay/closeorder", func(c *gin.Context) {

		closeOrderResult, err := order.CloseOrder(pay, map[string]string{
			"appid":        pay.Config.Appid,
			"mch_id":       pay.Config.Mchid,
			"nonce_str":    util.GetRandString(32),
			"out_trade_no": c.Request.URL.Query().Get("out_trade_no"),
		})

		fmt.Println(closeOrderResult, err)

		if err != nil {
			return
		}

		c.Writer.WriteString(fmt.Sprint(closeOrderResult))
	})
	// 下载交易账单
	router.GET("/api/wxpay/downloadbill", func(c *gin.Context) {

		downloadBill, err := download.DownloadBill(pay, map[string]string{
			"appid":     pay.Config.Appid,
			"mch_id":    pay.Config.Mchid,
			"nonce_str": util.GetRandString(32),
			"bill_date": c.Request.URL.Query().Get("date"),
			"bill_type": types.BillTypeALL,
		})

		fmt.Println(downloadBill, err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 下载资金账单
	router.GET("/api/wxpay/downloadfundflow", func(c *gin.Context) {

		downloadBill, err := download.DownloadFundFlow(pay, map[string]string{
			"appid":        pay.Config.Appid,
			"mch_id":       pay.Config.Mchid,
			"nonce_str":    util.GetRandString(32),
			"bill_date":    c.Request.URL.Query().Get("date"),
			"account_type": types.AccountTypeBasic,
			"sign_type":    types.SignTypeHMACSHA256,
		})

		fmt.Println(downloadBill, err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 下载评价数据
	router.GET("/api/wxpay/batchquerycomment", func(c *gin.Context) {

		downloadBill, err := download.BatchQueryComment(pay, map[string]string{
			"appid":      pay.Config.Appid,
			"mch_id":     pay.Config.Mchid,
			"nonce_str":  util.GetRandString(32),
			"sign_type":  types.SignTypeHMACSHA256,
			"begin_time": "20200724000000",
			"end_yime":   "20200812000000",
			"offset":     "0",
			"limit":      "100",
		})

		fmt.Println(string(downloadBill), err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 申请退款
	router.GET("/api/wxpay/refund", func(c *gin.Context) {

		result, err := refund.Refund(pay, map[string]string{
			"appid":         pay.Config.Appid,
			"mch_id":        pay.Config.Mchid,
			"nonce_str":     util.GetRandString(32),
			"out_trade_no":  "NO.10086",
			"out_refund_no": "NO.10086_REFUND",
			"total_fee":     "201",
			"refund_fee":    "201",
			"notify_url":    viper.GetString("REFUND_NOTIFY_URL"),
		})

		fmt.Println(result, err)

		if err != nil {
			return
		}
		c.Writer.WriteString(fmt.Sprint(result))
	})

	// 查询退款
	router.GET("/api/wxpay/refundquery", func(c *gin.Context) {

		result, err := refund.RefundQuery(pay, map[string]string{
			"appid":        pay.Config.Appid,
			"mch_id":       pay.Config.Mchid,
			"nonce_str":    util.GetRandString(32),
			"out_trade_no": "NO.10086",
		})

		fmt.Println(result, err)

		if err != nil {
			return
		}
		c.Writer.WriteString(fmt.Sprint(result))
	})

	// 短链接转换
	router.GET("/api/wxpay/shorturl", func(c *gin.Context) {

		result, err := native.ShortUrl(pay, map[string]string{
			"appid":     pay.Config.Appid,
			"mch_id":    pay.Config.Mchid,
			"nonce_str": util.GetRandString(32),
			"long_url":  c.Request.URL.Query().Get("url")})

		fmt.Println(result, err)

		if err != nil {
			return
		}
		c.Writer.WriteString(fmt.Sprint(result))
	})

	// 分账
	router.GET("/api/wxpay/profit_sharing", func(c *gin.Context) {

		result, err := profit_sharing.ProfitSharing(pay, map[string]string{
			"appid":          pay.Config.Appid,
			"mch_id":         pay.Config.Mchid,
			"nonce_str":      util.GetRandString(32),
			"sign_type":      types.SignTypeHMACSHA256,
			"transaction_id": "NO.10086",
			"out_order_no":   "NO.10086",
			"receivers": `[
			  {
				"type": "MERCHANT_ID",
				"account": "190001001",
				"amount": 100,
				"description": "分到商户"
			  },
			  {
				"type": "PERSONAL_WECHATID",
				"account": "86693952",
				"amount": 888,
				"description": "分到个人"
			  }
			]`,
		})

		fmt.Println(result, err)

		if err != nil {
			return
		}
		c.Writer.WriteString(fmt.Sprint(result))
	})

	// 企业付款 获取 rsa key
	router.GET("/api/wxpay/getpublickey", func(c *gin.Context) {

		result, err := corp_pay.GetPublicKey(pay, map[string]string{
			"mch_id":    pay.Config.Mchid,
			"nonce_str": util.GetRandString(32),
		})

		fmt.Println(result, err)

		if err != nil {
			return
		}
		c.Writer.WriteString(fmt.Sprint(result))
	})

	svr := &http.Server{
		Addr:    viper.GetString("LISTEN"),
		Handler: router,
	}

	go func() {
		err := svr.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	timeout := time.Duration(5) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := svr.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
