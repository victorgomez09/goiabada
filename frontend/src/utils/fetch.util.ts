import { FETCH } from "../constants/fetch.constant";

export const get = async <RESPONSE>(endpoint: string) => {
  const response = await fetch(`${FETCH.BASE_URL}${endpoint}`, {
    method: "GET",
  });
  const data = await response.json();
  return data as RESPONSE;
};

export const post = async <REQUEST, RESPONSE>(
  endpoint: string,
  body: REQUEST
) => {
  console.log(JSON.stringify(body));
  const response = await fetch(`${FETCH.BASE_URL}${endpoint}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await response.json();
  return data as RESPONSE;
};
