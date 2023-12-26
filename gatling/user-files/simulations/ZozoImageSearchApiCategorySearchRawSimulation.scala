import io.gatling.core.Predef._
import io.gatling.http.Predef._
import scala.concurrent.duration._

class ZozoImageSearchApiCategorySearchRawSimulation extends Simulation {

  val env = sys.env.getOrElse("ENV", "stg")
  val api_key = env match {
    case "prd" => "2d75d1fc-07a2-4de6-8fc9-46787d58742d"
    case "stg" => "80efafa7-e7db-4337-8eda-f60c2ed8947e"
    case "qa"  => "12c388bb-185c-4430-bc0b-31847d8aa13c"
    case "dev" => "fd1edd44-f5b3-401a-baf9-3cdedaa6410a"
  }
  val endpoint = env match {
    case "prd" => "https://api.zozo-image-search.com/"
    case "stg" => "https://api.zozo-image-search-stg.com/"
    case "qa"  => "https://api.zozo-image-search-qa.com/"
    case "dev" => "https://image-search-api-ext.mlops-dev.ml.zozo.com/"
  }
  val users_per_sec = sys.env.getOrElse("CONCURRENCY", "2").toInt
  val duration_sec = sys.env.getOrElse("DURATION", "10").toInt
  val nns_cache_hit_ratio = sys.env.getOrElse("NNS_CACHE_HIT_RATIO", "0.8").toDouble
  val feature_cache_hit_ratio = sys.env.getOrElse("FEATURE_CACHE_HIT_RATIO", "0.8").toDouble
  // choose image url which is included in feature cache if you want to measuer fature cache effectiveness.
  // ref. https://zozo.rickcloud.jp/wiki/pages/viewpage.action?pageId=90671532
  val image_url = sys.env.getOrElse("IMAGE_URL", "https://c.imgz.jp/290/53454290/53454290b_1_d_500.jpg")
  val category_id = sys.env.getOrElse("CATEGORY_ID", "2001")

  val httpProtocol = http
    .baseUrl(endpoint)
    .userAgentHeader("Mozilla/5.0 (Macintosh; Intel Mac OS X 10.8; rv:16.0) Gecko/20100101 Firefox/16.0")

  val headers = Map("Content-Type" -> "accept: application/json", "ImageSearch-Api-Key" -> api_key)
  val nns_cache_hit_request = exec(http("request category_search API")
    .get("/v1/category_search_raw")
    .queryParam("image_url", image_url)
    .queryParam("category_id", category_id)
    .queryParam("use_response_cache", false)
    .queryParam("use_nns_cache", true)
    .queryParam("use_feature_cache", false)
    .headers(headers)
    .check(status.is(200),
      bodyString.saveAs("ZOZO_IMAGE_RECOMMEND_BODY"),
      jsonPath("$.search_key").saveAs("search_key")))

  val single_nns_cache_hit_request = scenario("Request (" + env + ") category_search_raw  nns cache hit").exec(nns_cache_hit_request)

  val feature_cache_hit_request = exec(http("request category_search API")
    .get("/v1/category_search_raw")
    .queryParam("image_url", image_url)
    .queryParam("category_id", category_id)
    .queryParam("use_response_cache", false)
    .queryParam("use_nns_cache", false)
    .queryParam("use_feature_cache", true)
    .headers(headers)
    .check(status.is(200),
      bodyString.saveAs("ZOZO_IMAGE_RECOMMEND_BODY"),
      jsonPath("$.search_key").saveAs("search_key")))

  val single_feature_cache_hit_request = scenario("Request (" + env + ") category_search_raw feature cache hit").exec(feature_cache_hit_request)

  val nns_cache_miss_and_feature_cache_miss_request = exec(http("request category_search API")
    .get("/v1/category_search_raw")
    .queryParam("image_url", image_url)
    .queryParam("category_id", category_id)
    .queryParam("use_response_cache", false)
    .queryParam("use_nns_cache", false)
    .queryParam("use_feature_cache", false)
    .headers(headers)
    .check(status.is(200),
      bodyString.saveAs("ZOZO_IMAGE_RECOMMEND_BODY"),
      jsonPath("$.search_key").saveAs("search_key")))

  val single_nns_cache_miss_and_feature_cache_miss_request = scenario("Request (" + env + ") category_search_raw nns cache miss and feature cache miss").exec(nns_cache_miss_and_feature_cache_miss_request)

  val nns_cache_hit_per_sec = users_per_sec * nns_cache_hit_ratio
  val feature_cache_hit_per_sec = users_per_sec * feature_cache_hit_ratio
  val nns_cache_miss_and_feature_cache_miss_per_sec = users_per_sec * (1 - nns_cache_hit_ratio - feature_cache_hit_ratio)

  setUp(
    single_nns_cache_hit_request.inject(constantUsersPerSec(nns_cache_hit_per_sec) during(duration_sec seconds)).protocols(httpProtocol),
    single_feature_cache_hit_request.inject(constantUsersPerSec(feature_cache_hit_per_sec) during(duration_sec seconds)).protocols(httpProtocol),
    single_nns_cache_miss_and_feature_cache_miss_request.inject(constantUsersPerSec(nns_cache_miss_and_feature_cache_miss_per_sec) during(duration_sec seconds)).protocols(httpProtocol),
  )
}
