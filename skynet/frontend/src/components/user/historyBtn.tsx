import { checkPerm, getIntl, UserPerm } from '@/utils';
import { HistoryOutlined } from '@ant-design/icons';
import { useAccess } from '@umijs/max';
import { useState } from 'react';
import TableBtn from '../layout/table/tableBtn';
import HistoryModal from './historyModal';

export interface HistoryBtnProps {
  uid: string;
}

const HistoryBtn: React.FC<HistoryBtnProps> = (props) => {
  const [open, setOpen] = useState(false);
  const intl = getIntl();
  const access = useAccess();

  if (!checkPerm(access, 'user', UserPerm.PermRead))
    return <HistoryOutlined key="history" />;
  else
    return (
      <>
        <TableBtn
          icon={HistoryOutlined}
          tip={intl.get('pages.history.tip')}
          onClick={() => setOpen(true)}
        />
        <HistoryModal uid={props.uid} open={open} setOpen={setOpen} />
      </>
    );
};

export default HistoryBtn;
