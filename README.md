####业务和性能监控系统

#####框架：
1. statsd：数据收集。基于Node.Js
2. graphite：数据源，同时也可用作前端展示
3. grafana：前端展示

#####消息格式
`[company.]proj.module.func.<count | value | timing>`

* companye：公司名，如ezbuy或65daigou。作为公司内部使用，可省略以简化消息格式
* proj：项目名，如dgadmin，bulma，trending等
* module：模块名，项目中的具体模块
* func：模块完成的功能，或者需监控的切入点，推荐以具体功能作为名称

#####备注
统计分为步进(Count)，数值(Gauge)和时长(Timing)

#####应用场景
* 登录次数：应为步进统计，消息格式可为 `dgadmin.user.login.count`
* 商品价格：应为数值统计，消息格式可为 `ezbuy.goods.price.value`
* 下单时长：应为时长统计，消息格式可为 `ezbuy.order.place.timing`

#####使用示例

	import "spike/stats"

	func login(){
		// user login

		// becase the prefix (usually set as project name) is retrieved from config file
		// so we just set the module & function name here
		stats.Incr("user.login.count")
	}

	func getGoodsPrice() {
		// get goods price

		stats.FGauge("goods.price.value", 100.5)
		// or
		stats.Gauge("goods.price.value", 100)
	}

	func placeOrder() {
		t1 := stats.Now()

		// heavy work

		t2 := stats.Now()
		stats.TimingByValue("order.place.timing", t2.Sub(t1))
		// or
		stats.Timing("order.place.timing", t1, t2)
	}

#####后续
1. 时序数据库备份