import { UserPerm } from '@/utils';

export default function (initialState: {
  signin: boolean;
  permission: { [Key: string]: UserPerm };
  menu: [{ [Key: string]: any }];
}) {
  return initialState.permission;
}
