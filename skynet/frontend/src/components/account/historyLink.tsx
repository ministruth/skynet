import { FormattedMessage } from '@umijs/max';
import { Typography } from 'antd';
import { useState } from 'react';
import HistoryModal from '../user/historyModal';
const { Link } = Typography;

const HistoryLink = () => {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Link underline onClick={() => setOpen(true)}>
        <FormattedMessage id="pages.account.history" />
      </Link>
      <HistoryModal uid="self" open={open} setOpen={setOpen} />
    </>
  );
};

export default HistoryLink;
