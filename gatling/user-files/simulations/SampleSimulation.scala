import io.gatling.core.Predef._
import io.gatling.http.Predef._
import scala.concurrent.duration._

class SampleSimulation extends Simulation {

  val env = sys.env.getOrElse("ENV", "stg")
  val endpoint = env match {
    case "dev" => "https://sample-api/example.com/"
  }
  val users_per_sec = sys.env.getOrElse("CONCURRENCY", "2").toInt
  val duration_sec = sys.env.getOrElse("DURATION", "10").toInt

  val httpProtocol = http
    .baseUrl(endpoint)
    .userAgentHeader("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.8; rv:16.0) Gecko/20100101 Firefox/16.0")

  val headers = Map("Content-Type" -> "accept: application/json")
  val request = exec(http("request sample API")
    .get("/test")
    .headers(headers)
    .check(status.is(200)))

  val sample_request = scenario("Request (" + env + ") sample").exec(request)

  setUp(
    sample_request.inject(constantUsersPerSec(users_per_sec) during(duration_sec seconds)).protocols(httpProtocol),
  )
}
