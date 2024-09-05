import http from "k6/http";
import {
  randomString,
  randomIntBetween,
} from "https://jslib.k6.io/k6-utils/1.2.0/index.js";
import ws from "k6/ws";
import { check, sleep } from "k6";

export const options = {
  vus: 8192,
  iterations: 8192 * 16,
};

const sessionDuration = randomIntBetween(10000, 60000);

export default function () {
  const url = "ws://localhost:8080/ws";
  const params = { tags: { my_tag: "hello" } };

  const res = ws.connect(url, params, function (socket) {
    socket.on("open", () => {});
    // socket.on("message", (data) => console.log(`VU ${__VU}:`, "msg: ", data));
    socket.on("error", (e) => {
      if (e.error() != "websocket: close sent") {
        console.log(`VU ${__VU}:`, "An unexpected error occured: ", e.error());
      }
    });
    socket.setTimeout(function () {
      socket.close();
    }, sessionDuration);
  });

  check(res, { "status is 101": (r) => r && r.status === 101 });
}
