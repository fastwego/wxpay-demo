package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fastwego/wxpay/native"

	"github.com/fastwego/wxpay/refund"

	"github.com/fastwego/wxpay/download"

	"github.com/fastwego/wxpay"
	"github.com/fastwego/wxpay/order"
	"github.com/fastwego/wxpay/sandbox"
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
		//IsSandBoxMode: true,
		Cert: viper.GetString("CERT"),
	})

	// 开启沙箱模式
	if pay.Config.IsSandBoxMode {
		signKey, err := sandbox.GetSignKey(pay)
		if err == nil {
			pay.Config.ApiKey = signKey
		}
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

		unifiedOrderResult, err := order.UnifiedOrder(pay, order.UnifiedOrderParams{
			Body:           "BODY",
			OutTradeNo:     "NO.10086",
			TotalFee:       c.Request.URL.Query().Get("fee"),
			SPBillCreateIP: "12.123.14.223",
			NotifyURL:      viper.GetString("NOTIFYURL"),
			TradeType:      types.TradeTypeAPP,
		})
		fmt.Println(unifiedOrderResult, err)

		if err != nil {
			return
		}

		// 返回客户端预下单信息
		err = pay.Server.ResponsePaymentParams(c.Writer, c.Request, unifiedOrderResult.PrepayId)
	})

	// 查询订单
	router.GET("/api/wxpay/orderquery", func(c *gin.Context) {

		orderQueryResult, err := order.OrderQuery(pay, order.OrderQueryParams{
			OutTradeNo: c.Request.URL.Query().Get("out_trade_no"),
		})

		fmt.Println(orderQueryResult, err)

		if err != nil {
			return
		}

		data, err := xml.Marshal(orderQueryResult)
		c.Writer.Write(data)
	})

	// 关闭订单
	router.GET("/api/wxpay/closeorder", func(c *gin.Context) {

		closeOrderResult, err := order.CloseOrder(pay, order.CloseOrderParams{OutTradeNo: c.Request.URL.Query().Get("out_trade_no")})

		fmt.Println(closeOrderResult, err)

		if err != nil {
			return
		}

		data, err := xml.Marshal(closeOrderResult)
		c.Writer.Write(data)
	})
	// 下载交易账单
	router.GET("/api/wxpay/downloadbill", func(c *gin.Context) {

		downloadBill, err := download.DownloadBill(pay, download.DownloadBillParams{
			BillDate: c.Request.URL.Query().Get("date"),
			BillType: download.BillTypeALL,
			TarType:  "",
		})

		fmt.Println(downloadBill, err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 下载资金账单
	router.GET("/api/wxpay/downloadfundflow", func(c *gin.Context) {

		downloadBill, err := download.DownloadFundFlow(pay, download.DownloadFundFlowParams{
			BillDate:    c.Request.URL.Query().Get("date"),
			AccountType: download.AccountTypeBasic,
			TarType:     "",
		})

		fmt.Println(downloadBill, err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 下载评价数据
	router.GET("/api/wxpay/batchquerycomment", func(c *gin.Context) {

		downloadBill, err := download.BatchQueryComment(pay, download.BatchQueryCommentParams{
			BeginTime: "20200724000000",
			EndTime:   "20200812000000",
			Offset:    "0",
			Limit:     "100",
		})

		fmt.Println(string(downloadBill), err)

		if err != nil {
			return
		}

		_, _ = c.Writer.Write(downloadBill)
	})

	// 申请退款
	router.GET("/api/wxpay/refund", func(c *gin.Context) {

		refundResult, err := refund.Refund(pay, refund.RefundParams{
			OutTradeNo:  "NO.10086",
			OutRefundNo: "NO.10086_REFUND",
			TotalFee:    "201",
			RefundFee:   "201",
			NotifyUrl:   viper.GetString("REFUND_NOTIFY_URL"),
		})

		fmt.Println(refundResult, err)

		if err != nil {
			return
		}
		data, err := xml.Marshal(refundResult)
		c.Writer.Write(data)
	})

	// 查询退款
	router.GET("/api/wxpay/refundquery", func(c *gin.Context) {

		refundQueryResult, err := refund.RefundQuery(pay, refund.RefundQueryParams{
			OutTradeNo: "NO.10086",
		})

		fmt.Println(refundQueryResult, err)

		if err != nil {
			return
		}
		data, err := xml.Marshal(refundQueryResult)
		c.Writer.Write(data)
	})

	// 短链接转换
	router.GET("/api/wxpay/shorturl", func(c *gin.Context) {

		shortUrl, err := native.ShortUrl(pay, native.ShortUrlParams{LongUrl: c.Request.URL.Query().Get("url")})

		fmt.Println(shortUrl, err)

		if err != nil {
			return
		}
		data, err := xml.Marshal(shortUrl)
		c.Writer.Write(data)
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
