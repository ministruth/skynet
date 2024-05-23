import { v4 as uuidv4 } from 'uuid';

interface Message {}

export interface ShellConnect extends Message {
  id: String;
  type: 'ShellConnect';
  cmd: String;
  rows: number;
  cols: number;
}

export interface ShellInput extends Message {
  type: 'ShellInput';
  token: String;
  data: String;
}

export interface ShellResize extends Message {
  type: 'ShellResize';
  token: String;
  rows: number;
  cols: number;
}

export function newMessage(msg: Message) {
  return {
    id: uuidv4(),
    data: msg,
  };
}
