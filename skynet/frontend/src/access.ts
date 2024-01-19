import { GlobalState } from './app';

export default function (initialState: GlobalState) {
  if (initialState == undefined) return {};
  else return initialState.permission;
}
